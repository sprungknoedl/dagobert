package handler

import (
	"cmp"
	"encoding/csv"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/pkg/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type NoteCtrl struct {
	store model.NoteStore
}

func NewNoteCtrl(store model.NoteStore) *NoteCtrl {
	return &NoteCtrl{store}
}

func (ctrl NoteCtrl) List(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.store.FindNotes(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.NoteList(ctx(c), cid.String(), list))
}

func (ctrl NoteCtrl) Export(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := ctrl.store.ListNotes(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"templ.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Title", "Category", "Description"})
	for _, e := range list {
		w.Write([]string{
			e.ID.String(),
			e.Title,
			e.Category,
			e.Description,
		})
	}

	w.Flush()
	return nil
}

func (ctrl NoteCtrl) Import(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-notes", cid)
	now := time.Now()
	usr := c.Get("user").(string)

	return importHelper(c, uri, 4, func(c echo.Context, rec []string) error {
		id, err := ulid.Parse(rec[0])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Note{
			ID:           id,
			CaseID:       cid,
			Title:        rec[1],
			Category:     rec[2],
			Description:  rec[3],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = ctrl.store.SaveNote(cid, obj)
		return err
	})
}

func (ctrl NoteCtrl) View(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Note{CaseID: cid}
	if id != ZeroID {
		obj, err = ctrl.store.GetNote(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.NoteForm(ctx(c), templ.NoteDTO{
		ID:          id.String(),
		CaseID:      cid.String(),
		Title:       obj.Title,
		Category:    obj.Category,
		Description: obj.Description,
	}, valid.Result{}))
}

func (ctrl NoteCtrl) Save(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.NoteDTO{ID: id.String(), CaseID: cid.String()}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateNote(dto); !vr.Valid() {
		return render(c, templ.NoteForm(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := c.Get("user").(string)
	obj := model.Note{
		ID:           cmp.Or(id, ulid.Make()),
		CaseID:       cid,
		Title:        dto.Title,
		Category:     dto.Category,
		Description:  dto.Description,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != ZeroID {
		src, err := ctrl.store.GetNote(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := ctrl.store.SaveNote(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl NoteCtrl) Delete(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid note id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-note", cid, id) + "?confirm=yes"
		log.Printf("--> confirm uri: %s", uri)
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = ctrl.store.DeleteNote(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
