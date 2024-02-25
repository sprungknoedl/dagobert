package handler

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/doct"
)

var templates = map[string]doct.Template{}

type ReportCtrl struct{}

func NewReportCtrl() *ReportCtrl { return &ReportCtrl{} }

func LoadTemplates(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch filepath.Ext(path) {
		case ".ods":
			fallthrough
		case ".odp":
			fallthrough
		case ".odt":
			tpl, err := doct.LoadOdfTemplate(path)
			if err != nil {
				return err
			}

			templates[tpl.Name()] = tpl

		case ".docx":
			tpl, err := doct.LoadOxmlTemplate(path)
			if err != nil {
				return err
			}

			templates[tpl.Name()] = tpl
		}
		return nil
	})
}

func (ctrl ReportCtrl) List(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	list := apply2(templates, func(x doct.Template) string { return x.Name() })
	return render(c, templ.ReportList(ctx(c), cid, list))
}

func (ctrl ReportCtrl) Generate(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj, err := model.GetCaseFull(cid)
	if err != nil {
		return err
	}

	name := c.QueryParam("tpl")
	tpl, ok := templates[name]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid report template")
	}

	buf := new(bytes.Buffer)
	err = tpl.Render(buf, map[string]any{
		"Case": obj,
		"Now":  time.Now(),
	})
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s - %s%s", time.Now().Format("20060102"), obj.Name, tpl.Ext())
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Response().Header().Set("Content-Type", tpl.Type())
	c.Response().Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	c.Response().WriteHeader(http.StatusOK)

	_, err = io.Copy(c.Response().Writer, buf)
	return err
}
