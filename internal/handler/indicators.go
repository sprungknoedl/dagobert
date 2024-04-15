package handler

import (
	"cmp"
	"encoding/csv"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/pkg/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type IndicatorCtrl struct {
	store model.IndicatorStore
}

func NewIndicatorCtrl(store model.IndicatorStore) *IndicatorCtrl {
	return &IndicatorCtrl{store}
}

func (ctrl IndicatorCtrl) List(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.store.FindIndicators(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.IndicatorList(ctx(c), cid.String(), list))
}

func (ctrl IndicatorCtrl) Export(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := ctrl.store.ListIndicators(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"templ.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Type", "Value", "TLP", "Description", "Source"})
	for _, e := range list {
		w.Write([]string{
			e.ID.String(),
			e.Type,
			e.Value,
			e.TLP,
			e.Description,
			e.Source,
		})
	}

	w.Flush()
	return nil
}

func (ctrl IndicatorCtrl) Import(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-indicators", cid)
	now := time.Now()
	usr := c.Get("user").(string)

	return importHelper(c, uri, 6, func(c echo.Context, rec []string) error {
		id, err := ulid.Parse(cmp.Or(rec[0], ZeroID.String()))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Indicator{
			ID:           cmp.Or(id, ulid.Make()),
			CaseID:       cid,
			Type:         rec[1],
			Value:        rec[2],
			TLP:          rec[3],
			Description:  rec[4],
			Source:       rec[5],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = ctrl.store.SaveIndicator(cid, obj)
		return err
	})
}

func (ctrl IndicatorCtrl) Edit(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid indicator id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Indicator{CaseID: cid}
	if id != ZeroID {
		obj, err = ctrl.store.GetIndicator(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.IndicatorForm(ctx(c), templ.IndicatorDTO{
		ID:          id.String(),
		CaseID:      cid.String(),
		Type:        obj.Type,
		Value:       obj.Value,
		TLP:         obj.TLP,
		Description: obj.Description,
		Source:      obj.Source,
	}, valid.Result{}))
}

func (ctrl IndicatorCtrl) Save(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid indicator id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.IndicatorDTO{ID: id.String(), CaseID: cid.String()}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateIndicator(dto); !vr.Valid() {
		return render(c, templ.IndicatorForm(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := c.Get("user").(string)
	obj := model.Indicator{
		ID:           cmp.Or(id, ulid.Make()),
		CaseID:       cid,
		Type:         dto.Type,
		Value:        dto.Value,
		TLP:          dto.TLP,
		Description:  dto.Description,
		Source:       dto.Source,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != ZeroID {
		src, err := ctrl.store.GetIndicator(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := ctrl.store.SaveIndicator(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl IndicatorCtrl) Delete(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid indicator id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-indicator", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = ctrl.store.DeleteIndicator(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
