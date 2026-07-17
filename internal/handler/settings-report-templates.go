package handler

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) ReportTemplateList(w http.ResponseWriter, r *http.Request) {
	reports, err := h.Store.ListReportTemplates()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsReportTemplatesMany(h.Env(r), reports), nil)
}

func (h *Handler) ReportTemplateEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.ReportTemplate{ID: id}
	if id != "new" {
		var err error
		obj, err = h.Store.GetReportTemplate(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsReportTemplatesOne(h.Env(r), obj, valid.ValidationError{}), nil)
}

func (h *Handler) ReportTemplateSave(w http.ResponseWriter, r *http.Request) {
	// decode form
	dto := model.ReportTemplate{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateReportTemplate)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsReportTemplatesOne(h.Env(r), dto, vr), nil)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// get handle to form file
	fr, fh, err := r.FormFile("File")
	if err != nil && err != http.ErrMissingFile {
		Warn(w, r, err)
		return
	}

	new := dto.ID == "new"
	if err := resolveReportTemplateFile(h.Store, dto, new, fr, fh); err != nil {
		var terr templateError
		if errors.As(err, &terr) {
			Warn(w, r, err) // invalid template: user error, not a server fault
		} else {
			Err(w, r, err)
		}
		return
	}

	// finally save database object
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveReportTemplate(dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/report-templates/", nil)
}

// templateError marks a rejected report template (syntax or marker errors) so
// the handler can answer with a 400 instead of a 500.
type templateError struct{ error }

// resolveReportTemplateFile stores an uploaded report template (validating it and
// rolling a brand-new file back on rejection), or renames the stored template
// when an existing report changed its name. It is HTTP-free.
func resolveReportTemplateFile(store *model.Store, dto model.ReportTemplate, isNew bool, upload multipart.File, fh *multipart.FileHeader) error {
	if fh != nil && fh.Size > 0 {
		// prepare location for report storage
		dst := filepath.Join("files", BucketReportTemplates, dto.Name)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}

		// create or replace file; without O_TRUNC a smaller upload would
		// leave the tail of the previous template behind, corrupting the zip
		_, statErr := os.Stat(dst)
		existed := statErr == nil
		fw, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, upload)
		err = errors.Join(err, fw.Close())
		if err != nil {
			return err
		}

		// validate the uploaded template so marker reconstruction, hoisting
		// and template syntax errors surface now instead of at report time
		if _, err := LoadTemplate(dto.Name); err != nil {
			if !existed {
				os.Remove(dst)
			}
			return templateError{err}
		}

		// remove the previous template when the upload replaced it under a new name
		if !isNew {
			obj, err := store.GetReportTemplate(dto.ID)
			if err != nil {
				return err
			}
			if obj.Name != dto.Name {
				return os.Remove(filepath.Join("files", BucketReportTemplates, obj.Name))
			}
		}
		return nil
	}

	// no upload: a rename of an existing report moves the stored template along
	if !isNew {
		obj, err := store.GetReportTemplate(dto.ID)
		if err != nil {
			return err
		}
		if obj.Name != dto.Name {
			src := filepath.Join("files", BucketReportTemplates, obj.Name)
			dst := filepath.Join("files", BucketReportTemplates, dto.Name)
			return os.Rename(src, dst)
		}
	}
	return nil
}

func (h *Handler) ReportTemplateDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/report-templates/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	// try to delete file from disk
	obj, err := h.Store.GetReportTemplate(id)
	if err == nil {
		os.Remove(filepath.Join("files", "templates", obj.Name))
	}

	err = h.Store.DeleteReportTemplate(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/report-templates/", http.StatusSeeOther)
}
