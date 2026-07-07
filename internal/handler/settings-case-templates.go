package handler

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) CaseTemplateList(w http.ResponseWriter, r *http.Request) {
	templates, err := h.Store.ListTemplates()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsCaseTemplatesMany(h.Env(r), templates))
}

func (h *Handler) CaseTemplateEdit(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj := model.Case{ID: cid, IsTemplate: true}
	if cid != "new" {
		var err error
		obj, err = h.Store.GetCase(cid)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.CasesOne(h.Env(r), obj, valid.ValidationError{}))
}

func (h *Handler) CaseTemplateSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Case{ID: r.PathValue("cid")}
	err := Decode(h.Store, r, &dto, ValidateCase)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		dto.IsTemplate = true
		Render(w, r, http.StatusUnprocessableEntity, views.CasesOne(h.Env(r), dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	dto.IsTemplate = true
	if err := h.Store.SaveCase(dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/case-templates/")
}

func (h *Handler) CaseTemplateDelete(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := "/settings/case-templates/" + cid + "?confirm=yes"
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := h.Store.DeleteCase(cid); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/case-templates/", http.StatusSeeOther)
}

func (h *Handler) CaseTemplatePromoteForm(w http.ResponseWriter, r *http.Request) {
	cases, err := h.Store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsCaseTemplatesPromote(h.Env(r), cases))
}

func (h *Handler) CaseTemplatePromote(w http.ResponseWriter, r *http.Request) {
	form := struct {
		SourceID string
		Name     string
	}{}
	if err := Decode(h.Store, r, &form, nil); err != nil {
		Warn(w, r, err)
		return
	}

	src, err := h.Store.GetCase(form.SourceID)
	if err != nil {
		Err(w, r, err)
		return
	}

	dst := model.Case{
		ID:             fp.Random(10),
		Name:           form.Name,
		IsTemplate:     true,
		Classification: src.Classification,
		Severity:       src.Severity,
		Summary:        src.Summary,
	}
	if _, err := h.Store.CloneCaseContents(form.SourceID, dst); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/case-templates/", http.StatusSeeOther)
}
