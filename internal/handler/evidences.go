package handler

import (
	"crypto/sha1"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	cw.Write([]string{"ID", "Type", "Name", "Hash", "Size", "Notes", "StartsAt", "EndsAt", "Custom", "Fileless"})
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
			strconv.FormatBool(e.Fileless),
		})
	}

	cw.Flush()
}

func (h *Handler) EvidenceImport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/evidences/", cid)
	user := GetUser(r)
	h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 10, func(rec []string) {
			size, err := strconv.ParseInt(rec[4], 10, 64)
			if err != nil {
				Warn(w, r, err)
				return
			}

			fileless, err := strconv.ParseBool(rec[9])
			if err != nil {
				Warn(w, r, err)
				return
			}

			loc := filepath.Base(filepath.Clean(rec[2]))
			if !fileless {
				if _, err := os.Stat(filepath.Join("files", "evidences", cid, loc)); errors.Is(err, os.ErrNotExist) {
					Warn(w, r, err)
					return
				}
			}

			var startsAt model.Time
			if rec[6] != "" {
				t, err := time.Parse(time.RFC3339, rec[6])
				if err != nil {
					Warn(w, r, err)
					return
				} else {
					startsAt = model.Time(t)
				}
			}

			var endsAt model.Time
			if rec[7] != "" {
				t, err := time.Parse(time.RFC3339, rec[7])
				if err != nil {
					Warn(w, r, err)
					return
				} else {
					endsAt = model.Time(t)
				}
			}

			var custom model.Custom
			if err = custom.Scan(rec[8]); err != nil {
				Warn(w, r, err)
				return
			}

			obj := model.Evidence{
				ID:       fp.If(rec[0] == "", fp.Random(10), rec[0]),
				CaseID:   cid,
				Type:     rec[1],
				Name:     loc,
				Hash:     rec[3],
				Size:     size, // rec[4]
				Fileless: fileless,
				Notes:    rec[5],
				StartsAt: startsAt,
				EndsAt:   endsAt,
				Custom:   custom,
			}

			if err := tx.SaveEvidence(cid, obj); err != nil {
				Err(w, r, err)
				return
			}

			if err := tx.SaveEvidenceLog(cid, model.EvidenceLog{
				EvidenceID: obj.ID,
				Name:       obj.Name,
				User:       user.String(),
				Event:      model.EvidenceLogUploaded,
				Details:    "imported via CSV",
			}); err != nil {
				Err(w, r, err)
				return
			}
		})
	})
}

func (h *Handler) EvidenceEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	// new entries start fileless: the form must render the manual Hash/Size
	// inputs as editable and let the file picker adopt-or-attach on save.
	obj := model.Evidence{ID: id, CaseID: cid, Fileless: true}
	if id != "new" {
		var err error
		obj, err = h.Store.GetEvidence(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

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

	id := r.PathValue("id")
	cid := r.PathValue("cid")
	new := id == "new"
	if new {
		id = fp.Random(10)
	}

	// patch semantics: prefill dto from the stored record before Decode, so
	// both form and JSON decoding merge over current values — unsent fields
	// keep their value, an explicitly submitted "" clears
	dto := model.Evidence{ID: id, CaseID: cid}
	var old model.Evidence
	if !new {
		old, err = h.Store.GetEvidence(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
		dto = old
	}

	err = Decode(h.Store, r, &dto, ValidateEvidence)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr), vr)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// re-force path/served fields — no request body can flip them
	dto.ID = id
	dto.CaseID = cid
	if !new {
		dto.Fileless = old.Fileless
	} else {
		// creation infers the kind from upload presence; the attach routine
		// may still adopt a same-named disk file for a fileless creation
		dto.Fileless = fh == nil
	}

	if taken, err := h.Store.EvidenceNameTaken(cid, id, dto.Name); err != nil {
		Err(w, r, err)
		return
	} else if taken {
		vr := valid.ValidationError{"Name": valid.Condition{Name: "Name", Invalid: true,
			Message: "A file with this name already exists in this case."}}
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr), vr)
		return
	}

	// replacing the stored file of a file-backed entry is not supported; the
	// check stays here because its outcome is a rendered validation error,
	// an HTTP concern. Keyed on the stored record's Fileless flag (not on
	// dto.ID != "new") so an upload can still attach onto a fileless entry.
	if fh != nil && !new && !old.Fileless {
		vr := valid.ValidationError{"File": valid.Condition{Name: "File", Invalid: true,
			Message: "Replacing the file of an existing entry is not supported — create a new entry instead."}}
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr), vr)
		return
	}

	dto, attached, attachDetails, err := resolveEvidenceFile(dto, old, new, fr, fh)
	if vr, ok := err.(valid.ValidationError); ok {
		Render(w, r, http.StatusUnprocessableEntity, views.EvidencesOne(h.Env(r), dto, vr), vr)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// form-only: CollectCustom reads r.PostForm, which is empty for a JSON
	// request — assigning it unconditionally would wipe custom attributes
	// carried by a JSON patch
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		dto.Custom = CollectCustom(r)
	}

	user := GetUser(r)
	details := fp.If(new, dto.Hash, diffEvidence(old, dto))
	err = h.Store.Transaction(func(tx *model.Store) error {
		if err := tx.SaveEvidence(dto.CaseID, dto); err != nil {
			return err
		}

		if attached {
			if err := tx.SaveEvidenceLog(dto.CaseID, model.EvidenceLog{
				EvidenceID: dto.ID,
				Name:       dto.Name,
				User:       user.String(),
				Event:      model.EvidenceLogAttached,
				Details:    attachDetails,
			}); err != nil {
				return err
			}
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

// resolveEvidenceFile is HTTP-free (its validation errors are rendered by the
// caller) and splits on dto.Fileless, which the caller has already finalized:
// file-backed entries keep their stored metadata and only rename the disk
// file on a Name change (fresh file-backed creations write the upload
// instead, since there is no prior file to rename from); fileless entries run
// the attach routine, which may turn them file-backed. attached reports
// whether attach actually fired, for the caller's separate log entry.
func resolveEvidenceFile(dto, old model.Evidence, new bool, upload multipart.File, fh *multipart.FileHeader) (result model.Evidence, attached bool, details string, err error) {
	path := filepath.Join("files", "evidences", dto.CaseID, dto.Name)

	if !dto.Fileless {
		if new {
			dto.Hash, dto.Size, err = writeEvidenceFile(path, upload, fh.Size)
			return dto, false, "", err
		}

		// existing file-backed entry: Hash/Size are server-owned, always
		// re-forced from the stored record regardless of what was submitted
		dto.Hash, dto.Size = old.Hash, old.Size
		err = renameEvidenceFile(dto.CaseID, old.Name, dto.Name)
		return dto, false, "", err
	}

	return attachEvidenceFile(dto, path, upload, fh)
}

// writeEvidenceFile writes upload to path, failing if it already exists, and
// returns the SHA-1 hash and size of the copied bytes.
func writeEvidenceFile(path string, upload multipart.File, size int64) (hash string, _ int64, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", 0, err
	}
	fw, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if errors.Is(err, os.ErrExist) {
		return "", 0, valid.ValidationError{"Name": valid.Condition{Name: "Name", Invalid: true,
			Message: "A file with this name already exists in this case."}}
	} else if err != nil {
		return "", 0, err
	}
	defer func() { err = errors.Join(err, fw.Close()) }()

	hash, err = hashCopy(fw, upload)
	if err != nil {
		return "", 0, err
	}
	return hash, size, nil
}

// renameEvidenceFile moves a file-backed entry's disk file when its name
// changed. os.SameFile lets a case-only rename succeed on a case-insensitive
// filesystem (APFS), where stat-ing the destination resolves to the source
// file itself; any other existing file at the destination is a real
// collision. Other rename errors (e.g. the source file missing) propagate —
// a file-backed record whose disk file is gone is real corruption.
func renameEvidenceFile(cid, oldName, newName string) error {
	if oldName == newName {
		return nil
	}

	src := filepath.Join("files", "evidences", cid, oldName)
	dst := filepath.Join("files", "evidences", cid, newName)
	if dstStat, err := os.Stat(dst); err == nil {
		srcStat, err := os.Stat(src)
		if err != nil || !os.SameFile(dstStat, srcStat) {
			return valid.ValidationError{"Name": valid.Condition{Name: "Name", Invalid: true,
				Message: "A file with this name already exists in this case."}}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Rename(src, dst)
}

// attachEvidenceFile is the one sanctioned fileless → file-backed transition.
// It fires when an upload is present, or when a file already sits on disk
// under dto's name (adopt — the path for files too big to upload, copied
// server-side). Either way it writes/reads the file, computes its SHA-1,
// flips Fileless to false, and reports the entry's prior manual Hash/Size so
// the caller can log the custody-relevant overwrite. A fileless entry with
// neither trigger is returned unchanged — that is not an error.
func attachEvidenceFile(dto model.Evidence, path string, upload multipart.File, fh *multipart.FileHeader) (model.Evidence, bool, string, error) {
	prevHash, prevSize := dto.Hash, dto.Size

	if fh != nil {
		hash, size, err := writeEvidenceFile(path, upload, fh.Size)
		if err != nil {
			return dto, false, "", err
		}
		dto.Hash, dto.Size, dto.Fileless = hash, size, false
		return dto, true, fmt.Sprintf("recorded hash was %q, size %d", prevHash, prevSize), nil
	}

	fr, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return dto, false, "", nil
	} else if err != nil {
		return dto, false, "", err
	}
	defer fr.Close()

	stat, err := fr.Stat()
	if err != nil {
		return dto, false, "", err
	}
	hash, err := hashCopy(io.Discard, fr)
	if err != nil {
		return dto, false, "", err
	}

	dto.Hash, dto.Size, dto.Fileless = hash, stat.Size(), false
	return dto, true, fmt.Sprintf("recorded hash was %q, size %d", prevHash, prevSize), nil
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
// This is a display heuristic, not an equality check: it covers only today's
// editable fields, which for a fileless entry includes Hash/Size. Do not use
// an empty result to skip the SaveEvidence write itself.
func diffEvidence(old, new model.Evidence) string {
	changes := []string{}
	if old.Name != new.Name {
		changes = append(changes, fmt.Sprintf("name: %q → %q", old.Name, new.Name))
	}
	if old.Type != new.Type {
		changes = append(changes, "Type")
	}
	if old.Hash != new.Hash {
		changes = append(changes, "Hash")
	}
	if old.Size != new.Size {
		changes = append(changes, "Size")
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

	path := filepath.Join("files", "evidences", obj.CaseID, obj.Name)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	user := GetUser(r)
	if err := h.Store.SaveEvidenceLog(cid, model.EvidenceLog{
		EvidenceID: obj.ID,
		Name:       obj.Name,
		User:       user.String(),
		Event:      model.EvidenceLogDownloaded,
	}); err != nil {
		slog.Warn("failed to log evidence download", "err", err, "case", cid, "evidence", obj.ID)
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	// No explicit WriteHeader here: ServeFile picks the status itself
	// (200 / 206 Range / 304 conditional) and a premature 200 would override it.
	http.ServeFile(w, r, path)
}

func (h *Handler) EvidenceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/evidences/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	// try to delete file from disk — fileless entries have none, and today's
	// unconditional remove-by-name would risk deleting another record's file
	// on pre-uniqueness data that can still hold duplicate names
	obj, err := h.Store.GetEvidence(cid, id)
	if err == nil && !obj.Fileless {
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

	// Accept the confirm overlay explicitly instead of redirecting: a 303 to
	// the Evidence Log drawer's own URL would auto-accept via up-accept-location
	// and trigger dagobert.js's root-navigation handler, sending the drawer's
	// document to the root layer. The response body still renders the full
	// Evidence Log page so #list's up-hungry/up-if-layer="subtree" pair
	// (dagobert.js) picks up the refreshed rows in the still-open drawer.
	w.Header().Set("X-Up-Accept-Layer", `{"toast":"Evidence log purged."}`)

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
