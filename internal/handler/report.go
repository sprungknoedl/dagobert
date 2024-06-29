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

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
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
	list := fp.ToList(fp.ApplyM(templates, func(x doct.Template) string { return x.Name() }))
	Render(ctrl.store, w, r, http.StatusOK, "internal/views/reports-dialog.html", map[string]any{
		"list": list,
	})
}

func (ctrl ReportCtrl) Generate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")

	// ---
	// fetch data
	// ---
	var accerr error
	kase, err := ctrl.store.GetCase(cid)
	accerr = errors.Join(accerr, err)

	kase.Assets, err = ctrl.store.FindAssets(cid, "", "")
	accerr = errors.Join(accerr, err)

	kase.Events, err = ctrl.store.FindEvents(cid, "", "")
	accerr = errors.Join(accerr, err)

	kase.Evidences, err = ctrl.store.FindEvidences(cid, "", "")
	accerr = errors.Join(accerr, err)

	kase.Indicators, err = ctrl.store.FindIndicators(cid, "", "")
	accerr = errors.Join(accerr, err)

	kase.Malware, err = ctrl.store.FindMalware(cid, "", "")
	accerr = errors.Join(accerr, err)

	kase.Notes, err = ctrl.store.FindNotes(cid, "", "")
	accerr = errors.Join(accerr, err)

	kase.Tasks, err = ctrl.store.FindTasks(cid, "", "")
	accerr = errors.Join(accerr, err)

	if accerr != nil {
		Err(w, r, accerr)
		return
	}

	// ---
	// process report template
	// ---
	name := r.FormValue("Template")
	tpl, ok := templates[name]
	if !ok {
		Warn(w, r, errors.New("invalid report template"))
		return
	}

	buf := new(bytes.Buffer)
	err = tpl.Render(buf, map[string]any{
		"Case": kase,
		"Now":  time.Now(),
	})
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s%s", time.Now().Format("20060102"), kase.Name, tpl.Ext())
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", tpl.Type())
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)

	if _, err = io.Copy(w, buf); err != nil {
		Err(w, r, err)
	}
}
