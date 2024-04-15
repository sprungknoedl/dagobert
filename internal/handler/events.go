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

type EventCtrl struct {
	eventStore     model.EventStore
	assetStore     model.AssetStore
	indicatorStore model.IndicatorStore
}

func NewEventCtrl(assetStore model.AssetStore, eventStore model.EventStore, indicatorStore model.IndicatorStore) *EventCtrl {
	return &EventCtrl{
		eventStore:     eventStore,
		assetStore:     assetStore,
		indicatorStore: indicatorStore,
	}
}

func (ctrl EventCtrl) List(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.eventStore.FindEvents(cid, search, sort)
	if err != nil {
		return err
	}

	indicators, err := ctrl.indicatorStore.ListIndicators(cid)
	if err != nil {
		return err
	}

	return render(c, templ.EventList(ctx(c), cid.String(), list, indicators))
}

func (ctrl EventCtrl) Export(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := ctrl.eventStore.ListEvents(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"timeline.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Time", "Type", "Event System", "Direction", "Remote System", "Event", "Raw", "Key Event"})
	for _, e := range list {
		w.Write([]string{
			e.ID.String(),
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
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-events", cid)
	now := time.Now()
	usr := c.Get("user").(string)

	return importHelper(c, uri, 9, func(c echo.Context, rec []string) error {
		id, err := ulid.Parse(cmp.Or(rec[0], ZeroID.String()))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		t, err := time.Parse(time.RFC3339, rec[1])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		ke, err := strconv.ParseBool(cmp.Or(rec[8], "false"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Event{
			ID:           cmp.Or(id, ulid.Make()),
			CaseID:       cid,
			Time:         t,
			Type:         rec[2],
			AssetA:       rec[3],
			Direction:    rec[4],
			AssetB:       rec[5],
			Event:        rec[6],
			Raw:          rec[7],
			KeyEvent:     ke,
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = ctrl.eventStore.SaveEvent(cid, obj)
		return err
	})
}

func (ctrl EventCtrl) Edit(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Event{CaseID: cid}
	if id != ZeroID {
		obj, err = ctrl.eventStore.GetEvent(cid, id)
		if err != nil {
			return err
		}
	}

	assets, err := ctrl.assetStore.ListAssets(cid)
	if err != nil {
		return err
	}

	names := apply(assets, func(x model.Asset) string { return x.Name })
	return render(c, templ.EventForm(ctx(c), templ.EventDTO{
		ID:        obj.ID.String(),
		CaseID:    obj.CaseID.String(),
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
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.EventDTO{ID: id.String(), CaseID: cid.String()}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateEvent(dto); !vr.Valid() {
		assets, err := ctrl.assetStore.ListAssets(cid)
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
	usr := c.Get("user").(string)
	obj := model.Event{
		ID:           cmp.Or(id, ulid.Make()),
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

	if id != ZeroID {
		src, err := ctrl.eventStore.GetEvent(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := ctrl.eventStore.SaveEvent(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl EventCtrl) Delete(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-event", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = ctrl.eventStore.DeleteEvent(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
