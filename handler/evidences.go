package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/evidences"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/valid"
)

func ListEvidences(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	search := c.QueryParam("search")
	list, err := model.FindEvidences(cid, search)
	if err != nil {
		return err
	}

	return render(c, evidences.List(ctx(c), cid, list))
}

func ExportEvidences(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListEvidences(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"evidences.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Type", "Name", "Description", "Size", "Hash", "Location"})
	for _, e := range list {
		w.Write([]string{e.Type, e.Name, e.Description, strconv.FormatInt(e.Size, 10), e.Hash, e.Location})
	}

	w.Flush()
	return nil
}

func ImportEvidences(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-evidences", cid)
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 6, func(c echo.Context, rec []string) error {
		size, err := strconv.ParseInt(rec[3], 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Evidence{
			CaseID:       cid,
			Type:         rec[0],
			Name:         rec[1],
			Description:  rec[2],
			Size:         size,
			Hash:         rec[4],
			Location:     rec[5],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = model.SaveEvidence(cid, obj)
		return err
	})
}

func ViewEvidence(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Evidence{CaseID: cid}
	if id != 0 {
		obj, err = model.GetEvidence(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, evidences.Form(ctx(c), evidences.EvidenceDTO{
		ID:          id,
		CaseID:      cid,
		Type:        obj.Type,
		Name:        obj.Name,
		Description: obj.Description,
		Size:        obj.Size,
		Hash:        obj.Hash,
		Location:    obj.Location,
	}, valid.Result{}))
}

func SaveEvidence(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := evidences.EvidenceDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateEvidence(dto); !vr.Valid() {
		return render(c, evidences.Form(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Evidence{
		ID:           id,
		CaseID:       cid,
		Type:         dto.Type,
		Name:         dto.Name,
		Description:  dto.Description,
		Size:         dto.Size,
		Hash:         dto.Hash,
		Location:     dto.Location,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != 0 {
		src, err := model.GetEvidence(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveEvidence(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func DeleteEvidence(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-evidence", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteEvidence(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
