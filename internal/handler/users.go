package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type UserCtrl struct{}

func NewUserCtrl() *UserCtrl { return &UserCtrl{} }

func (ctrl UserCtrl) ListUsers(c echo.Context) error {
	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindUsers(search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.UserList(ctx(c), list))
}

func (ctrl UserCtrl) ViewUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid user id")
	}

	obj := model.User{}
	if id != 0 {
		obj, err = model.GetUser(id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.UserForm(ctx(c), templ.UserDTO{
		ID:      id,
		Name:    obj.Name,
		Company: obj.Company,
		Role:    obj.Role,
		Email:   obj.Email,
		Phone:   obj.Phone,
		Notes:   obj.Notes,
	}, valid.Result{}))
}

func (ctrl UserCtrl) SaveUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid user id")
	}

	dto := templ.UserDTO{ID: id}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateUser(dto); !vr.Valid() {
		return render(c, templ.UserForm(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.User{
		ID:           id,
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
		src, err := model.GetUser(id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveUser(obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl UserCtrl) DeleteUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid User id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-user", id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteUser(id)
	if err != nil {
		return err
	}

	return refresh(c)
}
