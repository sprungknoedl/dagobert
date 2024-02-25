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

type CaseCtrl struct{}

func NewCaseCtrl() *CaseCtrl { return &CaseCtrl{} }

func (ctrl CaseCtrl) List(c echo.Context) error {
	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindCases(search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.CaseList(ctx(c), list))
}

func (ctrl CaseCtrl) Export(c echo.Context) error {
	list, err := model.ListCases()
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"cases.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Name", "Severity", "Classification", "Closed", "Outcome", "Summary"})
	for _, e := range list {
		w.Write([]string{
			strconv.FormatInt(e.ID, 10),
			e.Name,
			e.Severity,
			e.Classification,
			strconv.FormatBool(e.Closed),
			e.Outcome,
			e.Summary,
		})
	}

	w.Flush()
	return nil
}

func (ctrl CaseCtrl) ImportCases(c echo.Context) error {
	uri := c.Echo().Reverse("import-cases")
	now := time.Now()
	usr := getUser(c)

	return importHelper(c, uri, 7, func(c echo.Context, rec []string) error {
		id, err := strconv.ParseInt(rec[0], 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		closed, err := strconv.ParseBool(rec[4])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Case{
			ID:             id,
			Name:           rec[1],
			Severity:       rec[2],
			Classification: rec[3],
			Closed:         closed,
			Outcome:        rec[5],
			Summary:        rec[6],
			DateAdded:      now,
			UserAdded:      usr,
			DateModified:   now,
			UserModified:   usr,
		}

		_, err = model.SaveCase(obj)
		return err
	})
}

func (ctrl CaseCtrl) Show(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj, err := model.GetCase(cid)
	if err != nil {
		return err
	}

	return render(c, templ.CaseOverview(ctx(c), obj))
}

func (ctrl CaseCtrl) Edit(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	var obj model.Case
	if cid != 0 {
		obj, err = model.GetCase(cid)
		if err != nil {
			return err
		}
	}

	vr := valid.Result{}
	return render(c, templ.CaseForm(ctx(c), templ.CaseDTO{
		ID:             obj.ID,
		Name:           obj.Name,
		Closed:         obj.Closed,
		Classification: obj.Classification,
		Severity:       obj.Severity,
		Outcome:        obj.Outcome,
		Summary:        obj.Summary,
	}, vr))
}

func (ctrl CaseCtrl) Save(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.CaseDTO{ID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateCase(dto); !vr.Valid() {
		return render(c, templ.CaseForm(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Case{
		ID:             cid,
		Name:           dto.Name,
		Closed:         dto.Closed,
		Classification: dto.Classification,
		Severity:       dto.Severity,
		Outcome:        dto.Outcome,
		Summary:        dto.Summary,
		DateAdded:      now,
		UserAdded:      usr,
		DateModified:   now,
		UserModified:   usr,
	}

	if cid != 0 {
		src, err := model.GetCase(cid)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveCase(obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl CaseCtrl) Delete(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-case", cid) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	if err := model.DeleteCase(cid); err != nil {
		return err
	}

	return refresh(c)
}
