package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type NoteCtrl struct {
	Ctrl
}

func NewNoteCtrl(store *model.Store, acl *auth.ACL) *NoteCtrl {
	return &NoteCtrl{BaseCtrl{store, acl}}
}

func (ctrl NoteCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListNotes(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.NotesMany(Env(ctrl, r), list))
}

func (ctrl NoteCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListNotes(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(ctrl.Store(), r)
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

func (ctrl NoteCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/notes/", cid)
	ctrl.Store().Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, ctrl.ACL(), w, r, uri, 5, func(rec []string) {
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
	})
}

func (ctrl NoteCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Note{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.Store().GetNote(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.NotesOne(Env(ctrl, r), obj, valid.ValidationError{}))
}

func (ctrl NoteCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Note{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err := Decode(ctrl.Store(), r, &dto, ValidateNote)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.NotesOne(Env(ctrl, r), dto, vr))
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
	if err := ctrl.Store().SaveNote(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/notes/", dto.CaseID))
}

func (ctrl NoteCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/notes/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := ctrl.Store().DeleteNote(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/notes/", cid), http.StatusSeeOther)
}
