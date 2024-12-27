package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type TaskCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewTaskCtrl(store *model.Store, acl *ACL) *TaskCtrl {
	return &TaskCtrl{store, acl}
}

func (ctrl TaskCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListTasks(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/tasks-many.html", map[string]any{
		"title": "Tasks",
		"rows":  list,
	})
}

func (ctrl TaskCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListTasks(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Tasks.csv", time.Now().Format("20060102"), GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Type", "Task", "Done", "Owner", "Due Date"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Task,
			strconv.FormatBool(e.Done),
			e.Owner,
			e.DateDue.Format(time.RFC3339),
		})
	}

	cw.Flush()
}

func (ctrl TaskCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/tasks/", cid)
	ImportCSV(ctrl.store, ctrl.acl, w, r, uri, 6, func(rec []string) {
		done, err := strconv.ParseBool(cmp.Or(rec[3], "false"))
		if err != nil {
			Warn(w, r, err)
		}

		datedue, err := time.Parse(time.RFC3339, cmp.Or(rec[5], ZeroTime.Format(time.RFC3339)))
		if err != nil {
			Warn(w, r, err)
		}

		obj := model.Task{
			ID:      rec[0],
			Type:    rec[1],
			Task:    rec[2],
			Done:    done, // 3
			Owner:   rec[4],
			DateDue: model.Time(datedue), // 5
			CaseID:  cid,
		}

		err = ctrl.store.SaveTask(cid, obj)
		Err(w, r, err)
	})
}

func (ctrl TaskCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Task{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetTask(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/tasks-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl TaskCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Task{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	if vr := ValidateTask(dto); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/tasks-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.ID = fp.If(dto.ID == "new", random(10), dto.ID)
	if err := ctrl.store.SaveTask(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/tasks/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl TaskCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/tasks/%s?confirm=yes", cid, id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	err := ctrl.store.DeleteTask(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/tasks/", cid), http.StatusSeeOther)
}
