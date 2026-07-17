package handler

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) NoteList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListNotes(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.NotesMany(h.Env(r), list), list)
}

func (h *Handler) NoteExport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListNotes(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Notes.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Title", "Category", "Description", "Custom"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Title,
			e.Category,
			e.Description,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (h *Handler) NoteImport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/notes/", cid)
	if err := h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 5, func(rec []string) {
			var custom model.Custom
			if len(rec) > 4 {
				custom.Scan(rec[4])
			}

			obj := model.Note{
				ID:          fp.If(rec[0] == "", fp.Random(10), rec[0]),
				Title:       rec[1],
				Category:    rec[2],
				Description: rec[3],
				CaseID:      cid,
				Custom:      custom,
			}

			if err := tx.SaveNote(cid, obj); err != nil {
				Err(w, r, err)
				return
			}
		})
	}); err != nil {
		// ImportCSV already wrote the HTTP response before Transaction() returns,
		// so a commit failure here can only be surfaced via logging.
		slog.Error("note import transaction failed to commit", "err", err, "case", cid)
	}
}

func (h *Handler) NoteEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Note{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = h.Store.GetNote(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.NotesOne(h.Env(r), obj, valid.ValidationError{}), obj)
}

func (h *Handler) NoteSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Note{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err := Decode(h.Store, r, &dto, ValidateNote)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.NotesOne(h.Env(r), dto, vr), vr)
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// NOTE: form-only for now — CollectCustom reads r.PostForm, so a JSON API
	// request yields an empty map and won't carry custom values.
	dto.Custom = CollectCustom(r)

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveNote(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/notes/", dto.CaseID), dto)
}

func (h *Handler) NoteDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/notes/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	err := h.Store.DeleteNote(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/notes/", cid), http.StatusSeeOther)
}
