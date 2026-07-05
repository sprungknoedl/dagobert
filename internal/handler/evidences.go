package handler

import (
	"crypto/sha1"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) EvidenceList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListEvidences(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.EvidencesMany(h.Env(r), list))
}

func (h *Handler) EvidenceExport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListEvidences(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Evidences.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Type", "Name", "Hash", "Size", "Notes", "StartsAt", "EndsAt", "Custom"})
	for _, e := range list {
		startsAt := ""
		if !e.StartsAt.IsZero() {
			startsAt = e.StartsAt.Format(time.RFC3339)
		}
		endsAt := ""
		if !e.EndsAt.IsZero() {
			endsAt = e.EndsAt.Format(time.RFC3339)
		}
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Name,
			e.Hash,
			strconv.FormatInt(e.Size, 10),
			e.Notes,
			startsAt,
			endsAt,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (h *Handler) EvidenceImport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/evidences/", cid)
	h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 0, func(rec []string) {
			size, err := strconv.ParseInt(rec[4], 10, 64)
			if err != nil {
				Warn(w, r, err)
				return
			}

			loc := filepath.Base(filepath.Clean(rec[2]))
			if _, err := os.Stat(filepath.Join("files", "evidences", cid, loc)); errors.Is(err, os.ErrNotExist) {
				Warn(w, r, err)
				return
			}

			var startsAt model.Time
			if len(rec) > 6 && rec[6] != "" {
				t, err := time.Parse(time.RFC3339, rec[6])
				if err != nil {
					Warn(w, r, err)
				} else {
					startsAt = model.Time(t)
				}
			}

			var endsAt model.Time
			if len(rec) > 7 && rec[7] != "" {
				t, err := time.Parse(time.RFC3339, rec[7])
				if err != nil {
					Warn(w, r, err)
				} else {
					endsAt = model.Time(t)
				}
			}

			var custom model.Custom
			if len(rec) > 8 {
				custom.Scan(rec[8])
			}

			obj := model.Evidence{
				ID:       fp.If(rec[0] == "", fp.Random(10), rec[0]),
				CaseID:   cid,
				Type:     rec[1],
				Name:     loc,
				Hash:     rec[3],
				Size:     size, // rec[4]
				Notes:    rec[5],
				StartsAt: startsAt,
				EndsAt:   endsAt,
				Custom:   custom,
			}

			if err := tx.SaveEvidence(cid, obj); err != nil {
				Err(w, r, err)
				return
			}
		})
	})
}

func (h *Handler) EvidenceEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Evidence{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = h.Store.GetEvidence(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	// assets, err := h.Store.ListAssets(cid)
	// if err != nil {
	// 	Err(w, r, err)
	// 	return
	// }

	Render(w, r, http.StatusOK, views.EvidencesOne(h.Env(r), obj, valid.ValidationError{}))
}

func (h *Handler) EvidenceSave(w http.ResponseWriter, r *http.Request) {
	// get handle to form file
	fr, fh, err := r.FormFile("File")
	if err != nil && err != http.ErrMissingFile {
		Warn(w, r, err)
		return
	}

	dto := model.Evidence{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err = Decode(h.Store, r, &dto, ValidateEvidence)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr))
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// replacing the stored file is not supported; the check stays here because
	// its outcome is a rendered validation error, an HTTP concern
	if fh != nil && fh.Size > 0 && dto.ID != "new" {
		vr := valid.ValidationError{"File": valid.Condition{Name: "File", Invalid: true,
			Message: "Replacing the file of an existing entry is not supported — create a new entry instead."}}
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr))
		return
	}

	dto, err = resolveEvidenceFile(h.Store, dto, fr, fh)
	if err != nil {
		Err(w, r, err)
		return
	}

	// NOTE: form-only for now — CollectCustom reads r.PostForm, so a JSON API
	// request yields an empty map and won't carry custom values.
	dto.Custom = CollectCustom(r)

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveEvidence(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	// trigger registered hooks
	if new {
		modules.TriggerOnEvidenceAdded(h.Store, dto)
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/evidences/", dto.CaseID))
}

// resolveEvidenceFile fills dto.Size and dto.Hash from the uploaded file, the
// existing DB record, or a file already present on disk (in that order). It is
// HTTP-free; the caller has already rejected uploads for existing entries.
func resolveEvidenceFile(store *model.Store, dto model.Evidence, upload multipart.File, fh *multipart.FileHeader) (model.Evidence, error) {
	dto.Name = filepath.Base(dto.Name) // sanitize name
	path := filepath.Join("files", "evidences", dto.CaseID, dto.Name)

	switch {
	case fh != nil && fh.Size > 0:
		// new upload: write to disk (fail if the file exists) and hash while writing
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return dto, err
		}
		fw, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return dto, err
		}
		defer fw.Close()
		if dto.Hash, err = hashCopy(fw, upload); err != nil {
			return dto, err
		}
		dto.Size = fh.Size

	case dto.ID != "new" && dto.Size > 0:
		// existing entry without new file: keep stored metadata
		obj, err := store.GetEvidence(dto.CaseID, dto.ID)
		if err != nil {
			return dto, err
		}
		dto.Size, dto.Hash = obj.Size, obj.Hash

	default:
		// adopt a file already present on disk; no file, no metadata — not an error
		dto.Size, dto.Hash = 0, ""
		fr, err := os.Open(path)
		if err != nil {
			return dto, nil
		}
		defer fr.Close()
		stat, err := fr.Stat()
		if err != nil {
			return dto, err
		}
		if dto.Hash, err = hashCopy(io.Discard, fr); err != nil {
			return dto, err
		}
		dto.Size = stat.Size()
	}
	return dto, nil
}

// hashCopy copies src to dst and returns the sha1 of the copied bytes.
func hashCopy(dst io.Writer, src io.Reader) (string, error) {
	hasher := sha1.New()
	if _, err := io.Copy(io.MultiWriter(dst, hasher), src); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (h *Handler) EvidenceDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := h.Store.GetEvidence(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	w.WriteHeader(http.StatusOK)
	http.ServeFile(w, r, filepath.Join("files", "evidences", obj.CaseID, obj.Name))
}

func (h *Handler) EvidenceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/evidences/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	// try to delete file from disk
	obj, err := h.Store.GetEvidence(cid, id)
	if err == nil {
		os.Remove(filepath.Join("files", "evidences", obj.CaseID, obj.Name))
	}

	err = h.Store.DeleteEvidence(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/evidences/", cid), http.StatusSeeOther)
}

func (h *Handler) EvidenceListModules(w http.ResponseWriter, r *http.Request) {
	ListModules(h, w, r, h.Store.GetEvidence)
}

func (h *Handler) EvidenceScheduleModule(w http.ResponseWriter, r *http.Request) {
	ScheduleModule(h, w, r, h.Store.GetEvidence)
}
