package handler

import (
	"cmp"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/pkg/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type TaskCtrl struct {
	store model.TaskStore
}

func NewTaskCtrl(store model.TaskStore) *TaskCtrl {
	return &TaskCtrl{store}
}

func (ctrl TaskCtrl) List(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.store.FindTasks(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.TaskList(ctx(c), cid.String(), list))
}

func (ctrl TaskCtrl) Export(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := ctrl.store.ListTasks(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"templ.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Type", "Task", "Done", "Owner", "Due Date"})
	for _, e := range list {
		w.Write([]string{
			e.ID.String(),
			e.Type,
			e.Task,
			strconv.FormatBool(e.Done),
			e.Owner,
			e.DateDue.Format(time.RFC3339),
		})
	}

	w.Flush()
	return nil
}

func (ctrl TaskCtrl) Import(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-tasks", cid)
	now := time.Now()
	usr := c.Get("user").(string)

	return importHelper(c, uri, 6, func(c echo.Context, rec []string) error {
		id, err := ulid.Parse(cmp.Or(rec[0], ZeroID.String()))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		done, err := strconv.ParseBool(cmp.Or(rec[3], "false"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		datedue, err := time.Parse(time.RFC3339, cmp.Or(rec[5], ZeroTime.Format(time.RFC3339)))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Task{
			ID:           cmp.Or(id, ulid.Make()),
			CaseID:       cid,
			Type:         rec[1],
			Task:         rec[2],
			Done:         done, // 3
			Owner:        rec[4],
			DateDue:      datedue, // 5
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = ctrl.store.SaveTask(cid, obj)
		return err
	})
}

func (ctrl TaskCtrl) Edit(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid task id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Task{CaseID: cid}
	if id != ZeroID {
		obj, err = ctrl.store.GetTask(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.TaskForm(ctx(c), templ.TaskDTO{
		ID:      id.String(),
		CaseID:  cid.String(),
		Type:    obj.Type,
		Task:    obj.Task,
		Done:    obj.Done,
		Owner:   obj.Owner,
		DateDue: formatNonZero("2006-01-02", obj.DateDue),
	}, valid.Result{}))
}

func (ctrl TaskCtrl) Save(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid task id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.TaskDTO{ID: id.String(), CaseID: cid.String()}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateTask(dto); !vr.Valid() {
		return render(c, templ.TaskForm(ctx(c), dto, vr))
	}

	dateDue, err := time.Parse("2006-01-02", dto.DateDue)
	if err != nil {
		return err // if ValidateTask is correct, this should never happen
	}

	now := time.Now()
	usr := c.Get("user").(string)
	obj := model.Task{
		ID:           cmp.Or(id, ulid.Make()),
		CaseID:       cid,
		Type:         dto.Type,
		Task:         dto.Task,
		Done:         dto.Done,
		Owner:        dto.Owner,
		DateDue:      dateDue,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != ZeroID {
		src, err := ctrl.store.GetTask(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := ctrl.store.SaveTask(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl TaskCtrl) Delete(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid task id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-task", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = ctrl.store.DeleteTask(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
