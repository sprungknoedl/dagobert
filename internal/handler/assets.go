package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type AssetCtrl struct{}

func NewAssetCtrl() *AssetCtrl { return &AssetCtrl{} }

func (ctrl AssetCtrl) List(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindAssets(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.AssetList(ctx(c), cid, list))
}

func (ctrl AssetCtrl) Export(c echo.Context) error {
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

func (ctrl AssetCtrl) Import(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-assets", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 6, func(c echo.Context, rec []string) error {
		analysed, err := strconv.ParseBool(rec[5])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Asset{
			CaseID:       cid,
			Type:         rec[0],
			Name:         rec[1],
			IP:           rec[2],
			Description:  rec[3],
			Compromised:  rec[4],
			Analysed:     analysed,
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveAsset(cid, obj)
		return err
	})
}

func (ctrl AssetCtrl) Edit(c echo.Context) error {
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

	return render(c, templ.AssetForm(ctx(c), templ.AssetDTO{
		ID:          id,
		CaseID:      cid,
		Type:        obj.Type,
		Name:        obj.Name,
		IP:          obj.IP,
		Description: obj.Description,
		Compromised: obj.Compromised,
		Analysed:    obj.Analysed,
	}, valid.Result{}))
}

func (ctrl AssetCtrl) Save(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.AssetDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateAsset(dto); !vr.Valid() {
		return render(c, templ.AssetForm(ctx(c), dto, vr))
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

func (ctrl AssetCtrl) Delete(c echo.Context) error {
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
