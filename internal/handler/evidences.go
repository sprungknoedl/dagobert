package handler

import (
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type EvidenceCtrl struct{}

func NewEvidenceCtrl() *EvidenceCtrl { return &EvidenceCtrl{} }

func (ctrl EvidenceCtrl) ListEvidences(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindEvidences(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.EvidenceList(ctx(c), cid, list))
}

func (ctrl EvidenceCtrl) ExportEvidences(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := model.ListEvidences(cid)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"templ.csv\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"Type", "Name", "Description", "Size", "Hash", "Location"})
	for _, e := range list {
		w.Write([]string{e.Type, e.Name, e.Description, strconv.FormatInt(e.Size, 10), e.Hash, e.Location})
	}

	w.Flush()
	return nil
}

func (ctrl EvidenceCtrl) ImportEvidences(c echo.Context) error {
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

func (ctrl EvidenceCtrl) ViewEvidence(c echo.Context) error {
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

	return render(c, templ.EvidenceForm(ctx(c), templ.EvidenceDTO{
		ID:          id,
		CaseID:      cid,
		Type:        obj.Type,
		Name:        obj.Name,
		Description: obj.Description,
	}, valid.Result{}))
}

func (ctrl EvidenceCtrl) DownloadEvidence(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj, err := model.GetEvidence(cid, id)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	c.Response().WriteHeader(http.StatusOK)
	return c.File(obj.Location)
}

func (ctrl EvidenceCtrl) SaveEvidence(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.EvidenceDTO{ID: id, CaseID: cid}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	dto.Name = filepath.Base(dto.Name) // sanitize name
	if vr := ValidateEvidence(dto); !vr.Valid() {
		return render(c, templ.EvidenceForm(ctx(c), dto, vr))
	}

	// get handle to form file
	fh, err := c.FormFile("file")
	if err != nil && err != http.ErrMissingFile {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// default values
	size := int64(0)
	hash := ""
	location := ""

	// process file if present
	if fh != nil && fh.Size > 0 {
		fr, err := fh.Open()
		if err != nil {
			return err
		}

		// prepare location for evidence storage
		location = filepath.Join("./files/evidences", strconv.FormatInt(cid, 10), dto.Name)
		err = os.MkdirAll(filepath.Dir(location), 0755)
		if err != nil {
			return err
		}

		// create file
		fw, err := os.Create(location)
		if err != nil {
			return err
		}

		// write and file and simultanously calculate sha1 hash
		hasher := sha1.New()
		mw := io.MultiWriter(fw, hasher)
		_, err = io.Copy(mw, fr)
		if err != nil {
			return err
		}

		size = fh.Size
		hash = fmt.Sprintf("%x", hasher.Sum(nil))
	} else if id != 0 {
		// keep metadata for existing evidences that did not change
		obj, err := model.GetEvidence(cid, id)
		if err != nil {
			return err
		}

		size = obj.Size
		hash = obj.Hash
		location = obj.Location
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.Evidence{
		ID:           id,
		CaseID:       cid,
		Type:         dto.Type,
		Name:         dto.Name,
		Description:  dto.Description,
		Size:         size,
		Hash:         hash,
		Location:     location,
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

func (ctrl EvidenceCtrl) DeleteEvidence(c echo.Context) error {
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
