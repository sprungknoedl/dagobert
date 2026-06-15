package handler

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := ctrl.Store().ListTemplates()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsTemplatesMany(Env(ctrl, r), templates))
}

func (ctrl SettingsCtrl) EditTemplate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj := model.Case{ID: cid, IsTemplate: true}
	if cid != "new" {
		var err error
		obj, err = ctrl.Store().GetCase(cid)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.CasesOne(Env(ctrl, r), obj, nil, "", false, nil, "", valid.ValidationError{}))
}

func (ctrl SettingsCtrl) SaveTemplate(w http.ResponseWriter, r *http.Request) {
	dto := model.Case{ID: r.PathValue("cid")}
	err := Decode(ctrl.Store(), r, &dto, ValidateCase)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		dto.IsTemplate = true
		Render(w, r, http.StatusUnprocessableEntity, views.CasesOne(Env(ctrl, r), dto, nil, "", false, nil, "", vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	dto.IsTemplate = true
	if err := ctrl.Store().SaveCase(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/templates/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := "/settings/templates/" + cid + "?confirm=yes"
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := ctrl.Store().DeleteCase(cid); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/templates/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) PromoteForm(w http.ResponseWriter, r *http.Request) {
	cases, err := ctrl.Store().ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsTemplatesPromote(Env(ctrl, r), cases))
}

func (ctrl SettingsCtrl) Promote(w http.ResponseWriter, r *http.Request) {
	form := struct {
		SourceID string
		Name     string
	}{}
	if err := Decode(ctrl.Store(), r, &form, nil); err != nil {
		Warn(w, r, err)
		return
	}

	src, err := ctrl.Store().GetCase(form.SourceID)
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
	if _, err := ctrl.Store().CloneCaseContents(form.SourceID, dst); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/templates/", http.StatusSeeOther)
}
