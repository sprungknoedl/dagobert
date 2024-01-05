package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/events"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
)

type EventDTO struct {
	Time      time.Time `form:"time"`
	Type      string    `form:"type"`
	AssetA    string    `form:"assetA"`
	AssetB    string    `form:"assetB"`
	Direction string    `form:"direction"`
	Event     string    `form:"event"`
	Raw       string    `form:"raw"`
	KeyEvent  bool      `form:"keyevent"`
}

func ListEvents(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	search := c.QueryParam("search")
	list, err := model.FindEvents(cid, search)
	if err != nil {
		return err
	}

	indicators, err := model.ListIndicators(cid)
	if err != nil {
		return err
	}

	return render(c, events.List(ctx(c), cid, list, indicators))
}

func ExportEvents(c echo.Context) error {
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

func ImportEvents(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-events", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, func(c echo.Context, rec []string) error {
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

func ViewEvent(c echo.Context) error {
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

	return render(c, events.Form(ctx(c), obj))
}

func SaveEvent(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := EventDTO{}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Event{
		ID:           id,
		CaseID:       cid,
		Time:         dto.Time,
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

func DeleteEvent(c echo.Context) error {
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
