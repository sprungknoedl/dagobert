package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type EventCtrl struct{}

func NewEventCtrl() *EventCtrl { return &EventCtrl{} }

func (ctrl EventCtrl) List(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindEvents(cid, search, sort)
	if err != nil {
		return err
	}

	indicators, err := model.ListIndicators(cid)
	if err != nil {
		return err
	}

	return render(c, templ.EventList(ctx(c), cid, list, indicators))
}

func (ctrl EventCtrl) Export(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListEvents(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"timeline.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Time", "Type", "Event System", "Direction", "Remote System", "Event", "Raw", "Key Event"})
	for _, e := range list {
		w.Write([]string{
			e.Time.Format(time.RFC3339),
			e.Type,
			e.AssetA,
			e.Direction,
			e.AssetB,
			e.Event,
			e.Raw,
			strconv.FormatBool(e.KeyEvent),
		})
	}

	w.Flush()
	return nil
}

func (ctrl EventCtrl) Import(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-events", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 8, func(c echo.Context, rec []string) error {
		t, err := time.Parse(time.RFC3339, rec[0])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		ke, err := strconv.ParseBool(rec[7])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Event{
			CaseID:       cid,
			Time:         t,
			Type:         rec[1],
			AssetA:       rec[2],
			Direction:    rec[3],
			AssetB:       rec[4],
			Event:        rec[5],
			Raw:          rec[6],
			KeyEvent:     ke,
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveEvent(cid, obj)
		return err
	})
}

func (ctrl EventCtrl) Show(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj, err := model.GetEvent(cid, id)
	if err != nil {
		return err
	}

	// related assets
	relatedAssets := []model.Asset{}
	if obj.AssetA != "" {
		x, err := model.GetAssetByName(cid, obj.AssetA)
		if err != nil {
			return err
		}

		relatedAssets = append(relatedAssets, x)
	}

	if obj.AssetB != "" && obj.AssetB != obj.AssetA {
		x, err := model.GetAssetByName(cid, obj.AssetB)
		if err != nil {
			return err
		}

		relatedAssets = append(relatedAssets, x)
	}

	// related indicators
	indicators, err := model.ListIndicators(cid)
	if err != nil {
		return err
	}

	search := obj.Event + obj.Raw
	relatedIndicators := []model.Indicator{}
	for _, x := range indicators {
		if strings.Contains(search, x.Value) {
			relatedIndicators = append(relatedIndicators, x)
		}
	}

	return render(c, templ.EventDetailsView(ctx(c), obj, relatedAssets, relatedIndicators))
}

func (ctrl EventCtrl) Edit(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Event{CaseID: cid}
	if id != 0 {
		obj, err = model.GetEvent(cid, id)
		if err != nil {
			return err
		}
	}

	assets, err := model.ListAssets(cid)
	if err != nil {
		return err
	}

	names := apply(assets, func(x model.Asset) string { return x.Name })
	return render(c, templ.EventForm(ctx(c), templ.EventDTO{
		ID:        obj.ID,
		CaseID:    obj.CaseID,
		Time:      formatNonZero(time.RFC3339, obj.Time),
		Type:      obj.Type,
		AssetA:    obj.AssetA,
		AssetB:    obj.AssetB,
		Direction: obj.Direction,
		Event:     obj.Event,
		Raw:       obj.Raw,
		KeyEvent:  obj.KeyEvent,
	}, names, valid.Result{}))
}

func (ctrl EventCtrl) Save(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.EventDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateEvent(dto); !vr.Valid() {
		assets, err := model.ListAssets(cid)
		if err != nil {
			return err
		}

		names := apply(assets, func(x model.Asset) string { return x.Name })
		return render(c, templ.EventForm(ctx(c), dto, names, vr))
	}

	t, err := time.Parse(time.RFC3339, dto.Time)
	if err != nil {
		return err // if ValidateEvent is correct, this should never happen
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Event{
		ID:           id,
		CaseID:       cid,
		Time:         t,
		Type:         dto.Type,
		AssetA:       dto.AssetA,
		AssetB:       dto.AssetB,
		Direction:    dto.Direction,
		Event:        dto.Event,
		Raw:          dto.Raw,
		KeyEvent:     dto.KeyEvent,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != 0 {
		src, err := model.GetEvent(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveEvent(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl EventCtrl) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-event", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteEvent(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
