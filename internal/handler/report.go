package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
	"github.com/sprungknoedl/dagobert/pkg/doct"
)

var templates = map[string]doct.Template{}

type ReportCtrl struct {
	store *model.Store
}

func NewReportCtrl(store *model.Store) *ReportCtrl {
	return &ReportCtrl{store}
}

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

func (ctrl ReportCtrl) List(w http.ResponseWriter, r *http.Request) {
	list := utils.ApplyM(templates, func(x doct.Template) string { return x.Name() })
	utils.Render(ctrl.store, w, r, "internal/views/reports-dialog.html", map[string]any{
		"list": list,
	})
}

func (ctrl ReportCtrl) Generate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj, err := ctrl.store.GetCaseFull(cid)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	name := r.URL.Query().Get("Template")
	tpl, ok := templates[name]
	if !ok {
		utils.Warn(w, r, errors.New("Invalid report template"))
		return
	}

	buf := new(bytes.Buffer)
	err = tpl.Render(buf, map[string]any{
		"Case": obj,
		"Now":  time.Now(),
	})
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s%s", time.Now().Format("20060102"), obj.Name, tpl.Ext())
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", tpl.Type())
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)

	if _, err = io.Copy(w, buf); err != nil {
		utils.Err(w, r, err)
	}
}
