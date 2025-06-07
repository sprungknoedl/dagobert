package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListReports(w http.ResponseWriter, r *http.Request) {
	reports, err := ctrl.store.ListReports()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "app/views/settings-reports-many.html", map[string]any{
		"title":   "Report Templates",
		"reports": reports,
	})
}

func (ctrl SettingsCtrl) EditReport(w http.ResponseWriter, r *http.Request) {
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

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "app/views/settings-reports-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl SettingsCtrl) SaveReport(w http.ResponseWriter, r *http.Request) {
	// decode form
	dto := model.Report{ID: r.PathValue("id")}
	if err := Decode(r, &dto); err != nil {
		Err(w, r, err)
		return
	}

	enums, err := ctrl.store.ListEnums()
	if err != nil {
		Err(w, r, err)
		return
	}

	// validate form
	dto.Name = filepath.Base(dto.Name) // sanitize name
	if vr := ValidateReport(dto, enums); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "app/views/settings-reports-one.html", map[string]any{
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
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.store.SaveReport(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/reports/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) DownloadReport(w http.ResponseWriter, r *http.Request) {
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

func (ctrl SettingsCtrl) DeleteReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/reports/%s?confirm=yes", id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "app/views/utils-confirm.html", map[string]any{
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

	http.Redirect(w, r, "/settings/reports/", http.StatusSeeOther)
}
