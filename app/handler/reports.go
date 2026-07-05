package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/doct"
)

const BucketReportTemplates = "templates"

func LoadTemplate(name string) (doct.Template, error) {
	// name is an untrusted form value joined into a filesystem path; require a
	// single flat element so it cannot traverse out of files/templates/ into
	// another case's evidence/malware dirs.
	if !isFlatName(name) {
		return nil, errors.New("invalid template")
	}

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

func (h *Handler) ReportDialog(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListReports()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.ReportsDialog(h.Env(r), list))
}

func (h *Handler) ReportGenerate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")

	// ---
	// fetch data
	// ---
	kase, err := h.Store.GetCaseFull(cid)
	if err != nil {
		Err(w, r, err)
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

	funcs := template.FuncMap{
		"note": func(title string) (string, error) {
			for _, n := range kase.Notes {
				if n.Title == title {
					return n.Description, nil
				}
			}
			return "", fmt.Errorf("report references a note titled %q, but no such note exists in this case", title)
		},
	}

	buf := new(bytes.Buffer)
	err = tpl.Render(buf, map[string]any{
		"Case": kase,
		"Now":  time.Now(),
	}, funcs)
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
