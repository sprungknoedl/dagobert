package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/doct"
)

const BucketReportTemplates = "templates"

func LoadTemplate(name string) (doct.Template, error) {
	path := filepath.Join("files", "templates", name)
	switch filepath.Ext(name) {
	case ".ods":
		fallthrough
	case ".odp":
		fallthrough
	case ".odt":
		return doct.LoadLibreTemplate(path)

	case ".docx":
		return doct.LoadMsTemplate(path)

	default:
		return nil, errors.New("invalid template")
	}
}

type ReportsCtrl struct {
	Ctrl
}

func NewReportsCtrl(store *model.Store, acl *ACL) *ReportsCtrl {
	return &ReportsCtrl{BaseCtrl{store, acl}}
}

func (ctrl ReportsCtrl) Dialog(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.Store().ListReports()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.ReportsDialog(Env(ctrl, r), list))
}

func (ctrl ReportsCtrl) Generate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")

	// ---
	// fetch data
	// ---
	var accerr error
	kase, err := ctrl.Store().GetCase(cid)
	accerr = errors.Join(accerr, err)

	kase.Assets, err = ctrl.Store().ListAssets(cid)
	accerr = errors.Join(accerr, err)

	kase.Events, err = ctrl.Store().ListEvents(cid)
	accerr = errors.Join(accerr, err)

	kase.Evidences, err = ctrl.Store().ListEvidences(cid)
	accerr = errors.Join(accerr, err)

	kase.Indicators, err = ctrl.Store().ListIndicators(cid)
	accerr = errors.Join(accerr, err)

	kase.Malware, err = ctrl.Store().ListMalware(cid)
	accerr = errors.Join(accerr, err)

	kase.Notes, err = ctrl.Store().ListNotes(cid)
	accerr = errors.Join(accerr, err)

	kase.Tasks, err = ctrl.Store().ListTasks(cid)
	accerr = errors.Join(accerr, err)

	if accerr != nil {
		Err(w, r, accerr)
		return
	}

	// ---
	// process report template
	// ---
	name := r.FormValue("Template")
	tpl, err := LoadTemplate(name)
	if err != nil {
		Warn(w, r, err)
		return
	}

	buf := new(bytes.Buffer)
	err = tpl.Render(buf, map[string]any{
		"Case": kase,
		"Now":  time.Now(),
	})
	if err != nil {
		Warn(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s%s", time.Now().Format("20060102"), kase.Name, tpl.Ext())
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", tpl.Type())
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)

	if _, err = io.Copy(w, buf); err != nil {
		Err(w, r, err)
		return
	}
}
