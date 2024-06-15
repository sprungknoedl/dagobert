package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type NoteCtrl struct {
	store *model.Store
}

func NewNoteCtrl(store *model.Store) *NoteCtrl {
	return &NoteCtrl{store}
}

func (ctrl NoteCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindNotes(cid, search, sort)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, "internal/views/notes-many.html", map[string]any{
		"title": "Notes",
		"rows":  list,
	})
}

func (ctrl NoteCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.FindNotes(cid, "", "")
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Notes.csv", time.Now().Format("20060102"), utils.GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Title", "Category", "Description"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Title,
			e.Category,
			e.Description,
		})
	}

	cw.Flush()
}

func (ctrl NoteCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := r.URL.RequestURI()
	ImportCSV(ctrl.store, w, r, uri, 4, func(rec []string) {
		obj := model.Note{
			ID:          rec[0],
			Title:       rec[1],
			Category:    rec[2],
			Description: rec[3],
			CaseID:      cid,
		}

		err := ctrl.store.SaveNote(cid, obj)
		utils.Err(w, r, err)
	})
}

func (ctrl NoteCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Note{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetNote(cid, id)
		if err != nil {
			utils.Err(w, r, err)
			return
		}
	}

	utils.Render(ctrl.store, w, r, "internal/views/notes-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl NoteCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Note{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := utils.Decode(r, &dto); err != nil {
		utils.Warn(w, r, err)
		return
	}

	if vr := ValidateNote(dto); !vr.Valid() {
		utils.Render(ctrl.store, w, r, "internal/views/notes-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.ID = utils.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveNote(dto.CaseID, dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}

func (ctrl NoteCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/notes/%s?confirm=yes", cid, id)
		utils.Render(ctrl.store, w, r, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
	}

	err := ctrl.store.DeleteNote(cid, id)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}
