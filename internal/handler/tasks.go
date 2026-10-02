package handler

import (
	"cmp"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

var taskCSV = views.CSVSpec{
	Columns: []string{"ID", "Type", "Task", "Done", "Owner", "Due Date", "Custom"},
	Sample:  []string{"", "Analysis", "A fictional task created from the CSV import sample.", "false", "", "2024-01-01T00:00:00Z", ""},
}

// resolveOwnerID validates the submitted owner id, if any, against assignable
// users — same rule as case assignees: an unknown or builtin user's id fails
// with "Invalid user." on Owner, an empty id means unassigned.
func resolveOwnerID(users []model.User, id *string) (*string, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	for _, u := range users {
		if u.ID == *id {
			return id, nil
		}
	}
	return nil, valid.ValidationError{"Owner": valid.Condition{Name: "Owner", Invalid: true, Message: "Invalid user."}}
}

func (h *Handler) TaskList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListTasks(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	comments, err := h.Store.CountComments(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.TasksMany(h.Env(r), list, comments), list)
}

func (h *Handler) TaskExport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListTasks(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Tasks.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write(taskCSV.Columns)
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Task,
			strconv.FormatBool(e.Done),
			e.Owner.Login,
			e.DateDue.Format(time.RFC3339),
			e.Custom.JSON(),
		})
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		slog.Error("failed to write task export csv", "err", err, "raddr", r.RemoteAddr, "case", cid)
	}
}

func (h *Handler) TaskImport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/tasks/", cid)

	// resolved once up front: an unknown or empty login never fails the row,
	// it just leaves the task unassigned, so CSVs from other instances import
	users, err := assignableUsers(h.Store)
	if err != nil {
		Err(w, r, err)
		return
	}

	ImportCSV(h.Store, w, r, uri, taskCSV, func(tx *model.Store, rec []string) error {
		done, err := strconv.ParseBool(cmp.Or(rec[3], "false"))
		if err != nil {
			return valid.ValidationError{"Done": valid.Condition{Name: "Done", Invalid: true, Message: err.Error()}}
		}

		datedue, err := time.Parse(time.RFC3339, cmp.Or(rec[5], time.Time{}.Format(time.RFC3339)))
		if err != nil {
			return valid.ValidationError{"DateDue": valid.Condition{Name: "DateDue", Invalid: true, Message: err.Error()}}
		}

		var custom model.Custom
		if len(rec) > 6 {
			if err := custom.Scan(rec[6]); err != nil {
				return valid.ValidationError{"Custom": valid.Condition{Name: "Custom", Invalid: true, Message: err.Error()}}
			}
		}

		var ownerID *string
		for _, u := range users {
			if u.Login == rec[4] {
				ownerID = &u.ID
				break
			}
		}

		obj := model.Task{
			ID:      fp.If(rec[0] == "", fp.Random(10), rec[0]),
			Type:    rec[1],
			Task:    rec[2],
			Done:    done, // 3
			OwnerID: ownerID,
			DateDue: model.Time(datedue), // 5
			CaseID:  cid,
			Custom:  custom,
		}

		return tx.SaveTask(cid, obj)
	})
}

func (h *Handler) TaskEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Task{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = h.Store.GetTask(cid, id)
		if errors.Is(err, model.ErrNotFound) {
			NotFound(w, r, err)
			return
		} else if err != nil {
			Err(w, r, err)
			return
		}
	}

	users, err := assignableUsers(h.Store)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.TasksOne(h.Env(r), obj, users, valid.ValidationError{}), obj)
}

func (h *Handler) TaskSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Task{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	decodeErr := Decode(h.Store, r, &dto, ValidateTask)

	users, err := assignableUsers(h.Store)
	if err != nil {
		Err(w, r, err)
		return
	}
	ownerID, ownerErr := resolveOwnerID(users, dto.OwnerID)

	if err := JoinV(decodeErr, ownerErr); err != nil {
		if vr, ok := err.(valid.ValidationError); ok {
			Render(w, r, http.StatusUnprocessableEntity, views.TasksOne(h.Env(r), dto, users, vr), vr)
			return
		}
		Warn(w, r, err)
		return
	}
	dto.OwnerID = ownerID

	dto.Custom = CollectCustom(r, dto.Custom)

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveTask(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/tasks/", dto.CaseID), dto)
}

func (h *Handler) TaskDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/tasks/%s?confirm=yes", cid, id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	err := h.Store.DeleteTask(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/tasks/", cid), http.StatusSeeOther)
}
