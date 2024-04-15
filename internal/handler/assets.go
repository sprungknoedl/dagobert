package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
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

type AssetCtrl struct {
	store model.AssetStore
}

func NewAssetCtrl(store model.AssetStore) *AssetCtrl {
	return &AssetCtrl{store}
}

func (ctrl AssetCtrl) List(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.store.FindAssets(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.AssetList(ctx(c), cid.String(), list))
}

func (ctrl AssetCtrl) Export(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := ctrl.store.ListAssets(cid)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s - %s - Assets.csv", time.Now().Format("20060102"), ctx(c).ActiveCase.Name)
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Type", "Name", "IP", "Description", "Compromised", "Analysed"})
	for _, e := range list {
		w.Write([]string{
			e.ID.String(),
			e.Type,
			e.Name,
			e.IP,
			e.Description,
			e.Compromised,
			strconv.FormatBool(e.Analysed),
		})
	}

	w.Flush()
	return nil
}

func (ctrl AssetCtrl) Import(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-assets", cid)
	now := time.Now()
	usr := c.Get("user").(string)

	return importHelper(c, uri, 7, func(c echo.Context, rec []string) error {
		id, err := ulid.Parse(cmp.Or(rec[0], ZeroID.String()))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		analysed, err := strconv.ParseBool(rec[6])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Asset{
			ID:           cmp.Or(id, ulid.Make()),
			CaseID:       cid,
			Type:         rec[1],
			Name:         rec[2],
			IP:           rec[3],
			Description:  rec[4],
			Compromised:  rec[5],
			Analysed:     analysed,
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = ctrl.store.SaveAsset(cid, obj)
		return err
	})
}

func (ctrl AssetCtrl) Edit(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Asset{CaseID: cid}
	if id != ZeroID {
		obj, err = ctrl.store.GetAsset(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.AssetForm(ctx(c), templ.AssetDTO{
		ID:          id.String(),
		CaseID:      cid.String(),
		Type:        obj.Type,
		Name:        obj.Name,
		IP:          obj.IP,
		Description: obj.Description,
		Compromised: obj.Compromised,
		Analysed:    obj.Analysed,
	}, valid.Result{}))
}

func (ctrl AssetCtrl) Save(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.AssetDTO{ID: id.String(), CaseID: cid.String()}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateAsset(dto); !vr.Valid() {
		return render(c, templ.AssetForm(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := c.Get("user").(string)
	obj := model.Asset{
		ID:           cmp.Or(id, ulid.Make()),
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

	if id != ZeroID {
		src, err := ctrl.store.GetAsset(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := ctrl.store.SaveAsset(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl AssetCtrl) Delete(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid asset id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-asset", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = ctrl.store.DeleteAsset(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
