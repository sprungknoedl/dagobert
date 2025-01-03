package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/doct"
	"github.com/sprungknoedl/dagobert/pkg/valid"
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
		return doct.LoadOdfTemplate(path)

	case ".docx":
		return doct.LoadOxmlTemplate(path)

	default:
		return nil, errors.New("invalid template")
	}
}

type ReportCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewReportCtrl(store *model.Store, acl *ACL) *ReportCtrl {
	return &ReportCtrl{store, acl}
}

func (ctrl ReportCtrl) Dialog(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.ListReports()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/reports-dialog.html", map[string]any{
		"rows": list,
	})
}

func (ctrl ReportCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.ListReports()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/reports-many.html", map[string]any{
		"title": "Report Templates",
		"rows":  list,
	})
}

func (ctrl ReportCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.Report{ID: id}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetReport(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/reports-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl ReportCtrl) Save(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.Report{ID: r.PathValue("id")}
	if err := Decode(r, &dto); err != nil {
		Err(w, r, err)
		return
	}

	// validate form
	dto.Name = filepath.Base(dto.Name) // sanitize name
	if vr := ValidateReport(dto); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/reports-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	// get handle to form file
	fr, fh, err := r.FormFile("File")
	if err != nil && err != http.ErrMissingFile {
		Warn(w, r, err)
		return
	}

	// process file if present
	new := dto.ID == "new"
	fileUpload := fh != nil && fh.Size > 0
	if fileUpload {
		// prepare location for Report storage
		dst := filepath.Join("files", BucketReportTemplates, dto.Name)
		err = os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			Err(w, r, err)
			return
		}

		// create file (but don't truncate if it exists)
		fw, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			Err(w, r, err)
			return
		}

		// write file
		_, err = io.Copy(fw, fr)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	// cleanup old file (if new report was uploaded)
	if fileUpload && !new {
		obj, err := ctrl.store.GetReport(dto.ID)
		if err != nil {
			Err(w, r, err)
			return
		}

		if obj.Name != dto.Name {
			path := filepath.Join("files", BucketReportTemplates, obj.Name)
			err = os.Remove(path)
			if err != nil {
				Err(w, r, err)
				return
			}
		}
	}

	// rename file
	if !fileUpload && !new {
		obj, err := ctrl.store.GetReport(dto.ID)
		if err != nil {
			Err(w, r, err)
			return
		}

		if obj.Name != dto.Name {
			src := filepath.Join("files", BucketReportTemplates, obj.Name)
			dst := filepath.Join("files", BucketReportTemplates, dto.Name)
			err = os.Rename(src, dst)
			if err != nil {
				Err(w, r, err)
				return
			}
		}
	}

	// finally save database object
	dto.ID = fp.If(new, random(10), dto.ID)
	if err := ctrl.store.SaveReport(dto); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "report:"+dto.ID, fp.If(new, "Added report template %q", "Updated report template %q"), dto.Name)
	http.Redirect(w, r, "/settings/reports/", http.StatusSeeOther)
}

func (ctrl ReportCtrl) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := ctrl.store.GetReport(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	w.WriteHeader(http.StatusOK)
	http.ServeFile(w, r, filepath.Join("files", "templates", obj.Name))
}

func (ctrl ReportCtrl) Generate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")

	// ---
	// fetch data
	// ---
	var accerr error
	kase, err := ctrl.store.GetCase(cid)
	accerr = errors.Join(accerr, err)

	kase.Assets, err = ctrl.store.ListAssets(cid)
	accerr = errors.Join(accerr, err)

	kase.Events, err = ctrl.store.ListEvents(cid)
	accerr = errors.Join(accerr, err)

	kase.Evidences, err = ctrl.store.ListEvidences(cid)
	accerr = errors.Join(accerr, err)

	kase.Indicators, err = ctrl.store.ListIndicators(cid)
	accerr = errors.Join(accerr, err)

	kase.Malware, err = ctrl.store.ListMalware(cid)
	accerr = errors.Join(accerr, err)

	kase.Notes, err = ctrl.store.ListNotes(cid)
	accerr = errors.Join(accerr, err)

	kase.Tasks, err = ctrl.store.ListTasks(cid)
	accerr = errors.Join(accerr, err)

	if accerr != nil {
		Err(w, r, accerr)
		return
	}

	// ---
	// process report template
	// ---
	name := r.FormValue("Template")
	obj, err := ctrl.store.GetReportByName(name)
	if err != nil {
		Err(w, r, err)
		return
	}

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

	Audit(ctrl.store, r, "report:"+obj.ID, "Generated report %q from template %q", filename, obj.Name)
}

func (ctrl ReportCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/reports/%s?confirm=yes", id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	// try to delete file from disk
	obj, err := ctrl.store.GetReport(id)
	if err == nil {
		os.Remove(filepath.Join("files", "templates", obj.Name))
	}

	err = ctrl.store.DeleteReport(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "report:"+obj.ID, "Deleted report template %q", obj.Name)
	http.Redirect(w, r, "/settings/reports/", http.StatusSeeOther)
}
