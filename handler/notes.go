package handler

import (
	"encoding/csv"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/notes"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
)

type NoteDTO struct {
	Title       string `form:"title"`
	Category    string `form:"category"`
	Description string `form:"description"`
}

func ListNotes(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	search := c.QueryParam("search")
	list, err := model.FindNotes(cid, search)
	if err != nil {
		return err
	}

	return render(c, notes.List(ctx(c), cid, list))
}

func ExportNotes(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListNotes(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"notes.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Title", "Category", "Description"})
	for _, e := range list {
		w.Write([]string{e.Title, e.Category, e.Description})
	}

	return nil
}

func ViewNote(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Note{CaseID: cid}
	if id != 0 {
		obj, err = model.GetNote(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, notes.Form(ctx(c), obj))
}

func SaveNote(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := NoteDTO{}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Note{
		ID:           id,
		CaseID:       cid,
		Title:        dto.Title,
		Category:     dto.Category,
		Description:  dto.Description,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != 0 {
		src, err := model.GetNote(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveNote(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func DeleteNote(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-note", cid, id) + "?confirm=yes"
		log.Printf("--> confirm uri: %s", uri)
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteNote(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
