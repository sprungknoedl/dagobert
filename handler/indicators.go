package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/indicators"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func ListIndicators(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindIndicators(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, indicators.List(ctx(c), cid, list))
}

func ExportIndicators(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListIndicators(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"indicators.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Type", "Value", "TLP", "Description", "Source"})
	for _, e := range list {
		w.Write([]string{e.Type, e.Value, e.TLP, e.Description, e.Source})
	}

	w.Flush()
	return nil
}

func ImportIndicators(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-indicators", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 5, func(c echo.Context, rec []string) error {
		obj := model.Indicator{
			CaseID:       cid,
			Type:         rec[0],
			Value:        rec[1],
			TLP:          rec[2],
			Description:  rec[3],
			Source:       rec[4],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveIndicator(cid, obj)
		return err
	})
}

func ViewIndicator(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid indicator id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Indicator{CaseID: cid}
	if id != 0 {
		obj, err = model.GetIndicator(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, indicators.Form(ctx(c), indicators.IndicatorDTO{
		ID:          id,
		CaseID:      cid,
		Type:        obj.Type,
		Value:       obj.Value,
		TLP:         obj.TLP,
		Description: obj.Description,
		Source:      obj.Source,
	}, valid.Result{}))
}

func SaveIndicator(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid indicator id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := indicators.IndicatorDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateIndicator(dto); !vr.Valid() {
		return render(c, indicators.Form(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Indicator{
		ID:           id,
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

	if id != 0 {
		src, err := model.GetIndicator(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveIndicator(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func DeleteIndicator(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid indicator id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-indicator", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteIndicator(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
