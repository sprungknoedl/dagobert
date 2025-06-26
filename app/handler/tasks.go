package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type TaskCtrl struct {
	Ctrl
}

func NewTaskCtrl(store *model.Store, acl *ACL) *TaskCtrl {
	return &TaskCtrl{BaseCtrl{store, acl}}
}

func (ctrl TaskCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListTasks(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.TasksMany(Env(ctrl, r), list))
}

func (ctrl TaskCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListTasks(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(ctrl.Store(), r)
	filename := fmt.Sprintf("%s - %s - Tasks.csv", time.Now().Format("20060102"), kase.Name)
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
	ImportCSV(ctrl.Store(), ctrl.ACL(), w, r, uri, 6, func(rec []string) {
		done, err := strconv.ParseBool(cmp.Or(rec[3], "false"))
		if err != nil {
			Warn(w, r, err)
		}

		datedue, err := time.Parse(time.RFC3339, cmp.Or(rec[5], ZeroTime.Format(time.RFC3339)))
		if err != nil {
			Warn(w, r, err)
		}

		obj := model.Task{
			ID:      fp.If(rec[0] == "", fp.Random(10), rec[0]),
			Type:    rec[1],
			Task:    rec[2],
			Done:    done, // 3
			Owner:   rec[4],
			DateDue: model.Time(datedue), // 5
			CaseID:  cid,
		}

		if err := ctrl.Store().SaveTask(cid, obj); err != nil {
			Err(w, r, err)
			return
		}
	})
}

func (ctrl TaskCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Task{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.Store().GetTask(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.TasksOne(Env(ctrl, r), obj, valid.Result{}))
}

func (ctrl TaskCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Task{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	enums, err := ctrl.Store().ListEnums()
	if err != nil {
		Err(w, r, err)
		return
	}

	if vr := ValidateTask(dto, enums); !vr.Valid() {
		Render(w, r, http.StatusUnprocessableEntity, views.TasksOne(Env(ctrl, r), dto, vr))
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveTask(dto.CaseID, dto); err != nil {
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
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := ctrl.Store().DeleteTask(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/tasks/", cid), http.StatusSeeOther)
}
