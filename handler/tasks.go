package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/tasks"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/valid"
)

func ListTasks(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindTasks(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, tasks.List(ctx(c), cid, list))
}

func ExportTasks(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListTasks(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"tasks.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Type", "Task", "Done", "Owner", "Due Date"})
	for _, e := range list {
		w.Write([]string{e.Type, e.Task, strconv.FormatBool(e.Done), e.Owner, e.DateDue.Format(time.RFC3339)})
	}

	w.Flush()
	return nil
}

func ImportTasks(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-tasks", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 5, func(c echo.Context, rec []string) error {
		done, err := strconv.ParseBool(rec[2])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		datedue, err := time.Parse(time.RFC3339, rec[4])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Task{
			CaseID:       cid,
			Type:         rec[0],
			Task:         rec[1],
			Done:         done, // 2
			Owner:        rec[3],
			DateDue:      datedue, // 4
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveTask(cid, obj)
		return err
	})
}

func ViewTask(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid task id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Task{CaseID: cid}
	if id != 0 {
		obj, err = model.GetTask(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, tasks.Form(ctx(c), tasks.TaskDTO{
		ID:      id,
		CaseID:  cid,
		Type:    obj.Type,
		Task:    obj.Task,
		Done:    obj.Done,
		Owner:   obj.Owner,
		DateDue: formatNonZero("2006-01-02", obj.DateDue),
	}, valid.Result{}))
}

func SaveTask(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid task id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := tasks.TaskDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateTask(dto); !vr.Valid() {
		return render(c, tasks.Form(ctx(c), dto, vr))
	}

	dateDue, err := time.Parse("2006-01-02", dto.DateDue)
	if err != nil {
		return err // if ValidateTask is correct, this should never happen
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Task{
		ID:           id,
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

	if id != 0 {
		src, err := model.GetTask(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveTask(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func DeleteTask(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid task id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-task", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteTask(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
