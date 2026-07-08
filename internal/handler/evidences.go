package handler

import (
	"crypto/sha1"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	comments, err := h.Store.CountComments(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.EvidencesMany(h.Env(r), list, comments), list)
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
	user := GetUser(r)
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

			tx.SaveEvidenceLog(cid, model.EvidenceLog{
				EvidenceID: obj.ID,
				Name:       obj.Name,
				User:       user.String(),
				Event:      model.EvidenceLogUploaded,
				Details:    "imported via CSV",
			})
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

	Render(w, r, http.StatusOK, views.EvidencesOne(h.Env(r), obj, valid.ValidationError{}), obj)
}

func (h *Handler) EvidenceSave(w http.ResponseWriter, r *http.Request) {
	// get handle to form file; JSON metadata-only saves carry no file, so a
	// multipart-less body (ErrNotMultipart) is tolerated like ErrMissingFile
	fr, fh, err := r.FormFile("File")
	if err != nil && err != http.ErrMissingFile && err != http.ErrNotMultipart {
		Warn(w, r, err)
		return
	}

	dto := model.Evidence{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err = Decode(h.Store, r, &dto, ValidateEvidence)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		if wantsJSON(r) {
			Render(w, r, http.StatusUnprocessableEntity, nil, vr)
			return
		}
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr), nil)
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
		if wantsJSON(r) {
			Render(w, r, http.StatusUnprocessableEntity, nil, vr)
			return
		}
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr), nil)
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

	var old model.Evidence
	if !new {
		old, err = h.Store.GetEvidence(dto.CaseID, dto.ID)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	user := GetUser(r)
	details := fp.If(new, dto.Hash, diffEvidence(old, dto))
	err = h.Store.Transaction(func(tx *model.Store) error {
		if err := tx.SaveEvidence(dto.CaseID, dto); err != nil {
			return err
		}
		if !new && details == "" {
			return nil // no-op edit — nothing changed, nothing to log
		}

		return tx.SaveEvidenceLog(dto.CaseID, model.EvidenceLog{
			EvidenceID: dto.ID,
			Name:       dto.Name,
			User:       user.String(),
			Event:      fp.If(new, model.EvidenceLogUploaded, model.EvidenceLogEdited),
			Details:    details,
		})
	})
	if err != nil {
		Err(w, r, err)
		return
	}

	// trigger registered automation rules
	if new {
		modules.TriggerOnEvidenceAdded(h.Store, dto)
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/evidences/", dto.CaseID), dto)
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

// diffEvidence returns the comma-separated list of changed field names between
// old and new, for the "edited" log entry. The rename is spelled out with its
// old/new values; every other field only names itself — the full values are
// deliberately not stored (bloat; would leak the archive password).
//
// This is a display heuristic, not an equality check: it skips Hash/Size
// (recomputed from disk, not user-edited) and only covers today's editable
// fields. Do not use an empty result to skip the SaveEvidence write itself.
func diffEvidence(old, new model.Evidence) string {
	changes := []string{}
	if old.Name != new.Name {
		changes = append(changes, fmt.Sprintf("name: %q → %q", old.Name, new.Name))
	}
	if old.Type != new.Type {
		changes = append(changes, "Type")
	}
	if old.Source != new.Source {
		changes = append(changes, "Source")
	}
	if old.Notes != new.Notes {
		changes = append(changes, "Notes")
	}
	if old.Password != new.Password {
		changes = append(changes, "Password")
	}
	if old.StartsAt.Format(time.RFC3339) != new.StartsAt.Format(time.RFC3339) {
		changes = append(changes, "StartsAt")
	}
	if old.EndsAt.Format(time.RFC3339) != new.EndsAt.Format(time.RFC3339) {
		changes = append(changes, "EndsAt")
	}
	if !maps.Equal(old.Custom, new.Custom) {
		changes = append(changes, "Custom")
	}
	return strings.Join(changes, ", ")
}

func (h *Handler) EvidenceDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := h.Store.GetEvidence(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	user := GetUser(r)
	h.Store.SaveEvidenceLog(cid, model.EvidenceLog{
		EvidenceID: obj.ID,
		Name:       obj.Name,
		User:       user.String(),
		Event:      model.EvidenceLogDownloaded,
	})

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	w.WriteHeader(http.StatusOK)
	http.ServeFile(w, r, filepath.Join("files", "evidences", obj.CaseID, obj.Name))
}

func (h *Handler) EvidenceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/evidences/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	// try to delete file from disk
	obj, err := h.Store.GetEvidence(cid, id)
	if err == nil {
		os.Remove(filepath.Join("files", "evidences", obj.CaseID, obj.Name))
	}

	user := GetUser(r)
	err = h.Store.DeleteEvidence(cid, id, user.String())
	if err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/evidences/", cid), http.StatusSeeOther)
}

func (h *Handler) EvidenceListModules(w http.ResponseWriter, r *http.Request) {
	ListModules(h, w, r, h.Store.GetEvidence)
}

func (h *Handler) EvidenceScheduleModule(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	ScheduleModule(h, w, r, h.Store.GetEvidence, func(tx *model.Store, cid, oid string, obj model.Evidence, module string) error {
		return tx.SaveEvidenceLog(cid, model.EvidenceLog{
			EvidenceID: oid,
			Name:       obj.Name,
			User:       user.String(),
			Event:      model.EvidenceLogModuleRun,
			Details:    module,
		})
	})
}

func (h *Handler) EvidenceLogList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	logs, err := h.Store.ListEvidenceLogs(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	evidences, err := h.Store.ListEvidences(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	live := fp.ToMap(evidences, func(e model.Evidence) string { return e.ID })
	Render(w, r, http.StatusOK, views.EvidenceLogsMany(h.Env(r), logs, live), logs)
}

func (h *Handler) EvidenceLogExport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListEvidenceLogs(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Evidence Log.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"Time", "Evidence", "Event", "User", "Details", "EvidenceID"})
	for _, l := range list {
		cw.Write([]string{
			l.Time.Format(time.RFC3339),
			l.Name,
			l.Event,
			l.User,
			l.Details,
			l.EvidenceID,
		})
	}
	cw.Flush()
}

// EvidenceLogPurge permanently removes all log rows for one deleted evidence.
// Administrator only — analysts must not be able to erase their own trail.
func (h *Handler) EvidenceLogPurge(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	eid := r.PathValue("eid")
	if GetUser(r).Role != "Administrator" {
		Forbidden(w, r)
		return
	}

	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/evidences/logs/%s?confirm=yes", cid, eid)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	if err := h.Store.PurgeEvidenceLogs(cid, eid); err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/evidences/logs", cid), http.StatusSeeOther)
}
