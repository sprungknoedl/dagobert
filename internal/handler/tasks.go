package handler

import (
	"cmp"
	"encoding/csv"
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
	cw.Write([]string{"ID", "Type", "Task", "Done", "Owner", "Due Date", "Custom"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Task,
			strconv.FormatBool(e.Done),
			e.Owner,
			e.DateDue.Format(time.RFC3339),
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (h *Handler) TaskImport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/tasks/", cid)
	if err := h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 7, func(rec []string) {
			done, err := strconv.ParseBool(cmp.Or(rec[3], "false"))
			if err != nil {
				Warn(w, r, err)
			}

			datedue, err := time.Parse(time.RFC3339, cmp.Or(rec[5], time.Time{}.Format(time.RFC3339)))
			if err != nil {
				Warn(w, r, err)
			}

			var custom model.Custom
			if len(rec) > 6 {
				if err := custom.Scan(rec[6]); err != nil {
					Warn(w, r, err)
					return
				}
			}

			obj := model.Task{
				ID:      fp.If(rec[0] == "", fp.Random(10), rec[0]),
				Type:    rec[1],
				Task:    rec[2],
				Done:    done, // 3
				Owner:   rec[4],
				DateDue: model.Time(datedue), // 5
				CaseID:  cid,
				Custom:  custom,
			}

			if err := tx.SaveTask(cid, obj); err != nil {
				Err(w, r, err)
				return
			}
		})
	}); err != nil {
		// ImportCSV already wrote the HTTP response before Transaction() returns,
		// so a commit failure here can only be surfaced via logging.
		slog.Error("task import transaction failed to commit", "err", err, "case", cid)
	}
}

func (h *Handler) TaskEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Task{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = h.Store.GetTask(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.TasksOne(h.Env(r), obj, valid.ValidationError{}), obj)
}

func (h *Handler) TaskSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Task{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err := Decode(h.Store, r, &dto, ValidateTask)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.TasksOne(h.Env(r), dto, vr), vr)
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
