package handler

import (
	"cmp"
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
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/pkg/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type EvidenceCtrl struct {
	store model.EvidenceStore
}

func NewEvidenceCtrl(store model.EvidenceStore) *EvidenceCtrl {
	return &EvidenceCtrl{store}
}

func (ctrl EvidenceCtrl) List(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.store.FindEvidences(cid, search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.EvidenceList(ctx(c), cid.String(), list))
}

func (ctrl EvidenceCtrl) Export(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list, err := ctrl.store.ListEvidences(cid)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s - %s - Evidences.csv", time.Now().Format("20060102"), ctx(c).ActiveCase.Name)
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Response().WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Response().Writer)
	w.Write([]string{"ID", "Type", "Name", "Description", "Size", "Hash", "Location"})
	for _, e := range list {
		w.Write([]string{
			e.ID.String(),
			e.Type,
			e.Name,
			e.Description,
			strconv.FormatInt(e.Size, 10),
			e.Hash,
			e.Location,
		})
	}

	w.Flush()
	return nil
}

func (ctrl EvidenceCtrl) Import(c echo.Context) error {
	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	uri := c.Echo().Reverse("import-evidences", cid)
	now := time.Now()
	usr := c.Get("user").(string)

	return importHelper(c, uri, 7, func(c echo.Context, rec []string) error {
		id, err := ulid.Parse(cmp.Or(rec[0], ZeroID.String()))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		size, err := strconv.ParseInt(rec[4], 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		obj := model.Evidence{
			ID:           cmp.Or(id, ulid.Make()),
			CaseID:       cid,
			Type:         rec[1],
			Name:         rec[2],
			Description:  rec[3],
			Size:         size,
			Hash:         rec[5],
			Location:     rec[7],
			DateAdded:    now,
			UserAdded:    usr,
			DateModified: now,
			UserModified: usr,
		}

		_, err = ctrl.store.SaveEvidence(cid, obj)
		return err
	})
}

func (ctrl EvidenceCtrl) Edit(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj := model.Evidence{CaseID: cid}
	if id != ZeroID {
		obj, err = ctrl.store.GetEvidence(cid, id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.EvidenceForm(ctx(c), templ.EvidenceDTO{
		ID:          id.String(),
		CaseID:      cid.String(),
		Type:        obj.Type,
		Name:        obj.Name,
		Description: obj.Description,
	}, valid.Result{}))
}

func (ctrl EvidenceCtrl) Download(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj, err := ctrl.store.GetEvidence(cid, id)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	c.Response().WriteHeader(http.StatusOK)
	return c.File(obj.Location)
}

func (ctrl EvidenceCtrl) Save(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	dto := templ.EvidenceDTO{ID: id.String(), CaseID: cid.String()}
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
		location = filepath.Join("./files/evidences", cid.String(), dto.Name)
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
	} else if id != ZeroID {
		// keep metadata for existing evidences that did not change
		obj, err := ctrl.store.GetEvidence(cid, id)
		if err != nil {
			return err
		}

		size = obj.Size
		hash = obj.Hash
		location = obj.Location
	}

	now := time.Now()
	usr := c.Get("user").(string)
	obj := model.Evidence{
		ID:           cmp.Or(id, ulid.Make()),
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

	if id != ZeroID {
		src, err := ctrl.store.GetEvidence(cid, id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := ctrl.store.SaveEvidence(cid, obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl EvidenceCtrl) Delete(c echo.Context) error {
	id, err := ulid.Parse(c.Param("id"))
	if err != nil || id == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid event id")
	}

	cid, err := ulid.Parse(c.Param("cid"))
	if err != nil || cid == ZeroID {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-evidence", cid, id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = ctrl.store.DeleteEvidence(cid, id)
	if err != nil {
		return err
	}

	return refresh(c)
}
