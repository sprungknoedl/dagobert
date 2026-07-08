package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) AutomationRuleList(w http.ResponseWriter, r *http.Request) {
	rules, err := h.Store.ListAutomationRules()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsAutomationRulesMany(h.Env(r), rules), nil)
}

func (h *Handler) AutomationRuleEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.AutomationRule{ID: id}
	if id != "new" {
		var err error
		obj, err = h.Store.GetHook(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	item := fp.ToList(fp.ApplyM(modules.Modules, func(m model.Module) model.ValueListItem { return model.ValueListItem{Name: m.Name()} }))
	Render(w, r, http.StatusOK, views.SettingsAutomationRulesOne(h.Env(r), obj, item, valid.ValidationError{}), nil)
}

func (h *Handler) AutomationRuleSave(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.AutomationRule{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateAutomationRule)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		item := fp.ToList(fp.ApplyM(modules.Modules, func(m model.Module) model.ValueListItem { return model.ValueListItem{Name: m.Name()} }))
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsAutomationRulesOne(h.Env(r), dto, item, vr), nil)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveHook(dto); err != nil {
		Err(w, r, err)
		return
	}

	// reload rules
	modules.LoadAutomationRules(h.Store)

	RedirectAfterSave(w, r, "/settings/automation-rules/", nil)
}

func (h *Handler) AutomationRuleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/automation-rules/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	err := h.Store.DeleteHook(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	// reload rules
	modules.LoadAutomationRules(h.Store)
	http.Redirect(w, r, "/settings/automation-rules/", http.StatusSeeOther)
}
