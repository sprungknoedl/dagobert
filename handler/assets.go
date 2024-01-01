package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/assets"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
)

type AssetCTO struct {
	Type        string `form:"type"`
	Name        string `form:"name"`
	IP          string `form:"ip"`
	Description string `form:"description"`
	Compromised string `form:"compromised"`
	Analysed    bool   `form:"analysed"`
}

func ListAssets(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	search := c.QueryParam("search")
	list, err := model.FindAssets(cid, search)
	if err != nil {
		return err
	}

	return render(c, assets.List(ctx(c), cid, list))
}

func ExportAssets(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListAssets(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"assets.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Type", "Name", "IP", "Description", "Compromised", "Analysed"})
	for _, e := range list {
		w.Write([]string{e.Type, e.Name, e.IP, e.Description, e.Compromised, strconv.FormatBool(e.Analysed)})
	}

	w.Flush()
	return nil
}

func ViewAsset(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Asset{CaseID: cid}
	if id != 0 {
		obj, err = model.GetAsset(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, assets.Form(ctx(c), obj))
}

func SaveAsset(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := AssetCTO{}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Asset{
		ID:           id,
		CaseID:       cid,
		Type:         dto.Type,
		Name:         dto.Name,
		IP:           dto.IP,
		Description:  dto.Description,
		Compromised:  dto.Compromised,
		Analysed:     dto.Analysed,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != 0 {
		src, err := model.GetAsset(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveAsset(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func DeleteAsset(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-asset", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteAsset(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
