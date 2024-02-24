package handler

import (
	"encoding/csv"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type NoteCtrl struct{}

func NewNoteCtrl() *NoteCtrl { return &NoteCtrl{} }

func (ctrl NoteCtrl) ListNotes(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindNotes(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.NoteList(ctx(c), cid, list))
}

func (ctrl NoteCtrl) ExportNotes(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListNotes(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"templ.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Title", "Category", "Description"})
	for _, e := range list {
		w.Write([]string{e.Title, e.Category, e.Description})
	}

	w.Flush()
	return nil
}

func (ctrl NoteCtrl) ImportNotes(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-notes", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 3, func(c echo.Context, rec []string) error {
		obj := model.Note{
			CaseID:       cid,
			Title:        rec[0],
			Category:     rec[1],
			Description:  rec[2],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveNote(cid, obj)
		return err
	})
}

func (ctrl NoteCtrl) ViewNote(c echo.Context) error {
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

	return render(c, templ.NoteForm(ctx(c), templ.NoteDTO{
		ID:          id,
		CaseID:      cid,
		Title:       obj.Title,
		Category:    obj.Category,
		Description: obj.Description,
	}, valid.Result{}))
}

func (ctrl NoteCtrl) SaveNote(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.NoteDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateNote(dto); !vr.Valid() {
		return render(c, templ.NoteForm(ctx(c), dto, vr))
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

func (ctrl NoteCtrl) DeleteNote(c echo.Context) error {
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
