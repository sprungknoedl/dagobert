package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/users"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/valid"
)

func ListUsers(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindUsers(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, users.List(ctx(c), cid, list))
}

func ExportUsers(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListUsers(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"users.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Name", "Company", "Role", "Email", "Phone", "Notes"})
	for _, e := range list {
		w.Write([]string{e.Name, e.Company, e.Role, e.Email, e.Phone, e.Notes})
	}

	w.Flush()
	return nil
}

func ImportUsers(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-users", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 6, func(c echo.Context, rec []string) error {
		obj := model.User{
			CaseID:       cid,
			Name:         rec[0],
			Company:      rec[1],
			Role:         rec[2],
			Email:        rec[3],
			Phone:        rec[4],
			Notes:        rec[5],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveUser(cid, obj)
		return err
	})
}

func ViewUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid user id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.User{CaseID: cid}
	if id != 0 {
		obj, err = model.GetUser(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, users.Form(ctx(c), users.UserDTO{
		ID:      id,
		CaseID:  cid,
		Name:    obj.Name,
		Company: obj.Company,
		Role:    obj.Role,
		Email:   obj.Email,
		Phone:   obj.Phone,
		Notes:   obj.Notes,
	}, valid.Result{}))
}

func SaveUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid user id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := users.UserDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateUser(dto); !vr.Valid() {
		return render(c, users.Form(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.User{
		ID:           id,
		CaseID:       cid,
		Name:         dto.Name,
		Company:      dto.Company,
		Role:         dto.Role,
		Email:        dto.Email,
		Phone:        dto.Phone,
		Notes:        dto.Notes,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != 0 {
		src, err := model.GetUser(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveUser(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func DeleteUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid User id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-user", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteUser(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
