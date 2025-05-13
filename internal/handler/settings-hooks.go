package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListHooks(w http.ResponseWriter, r *http.Request) {
	hooks, err := ctrl.store.ListHooks()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-hooks-many.html", map[string]any{
		"title": "Automation Rules",
		"hooks": hooks,
	})
}

func (ctrl SettingsCtrl) EditHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.Hook{ID: id}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetHook(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-hooks-one.html", map[string]any{
		"obj":   obj,
		"mods":  worker.List,
		"valid": valid.Result{},
	})
}

func (ctrl SettingsCtrl) SaveHook(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.Hook{ID: r.PathValue("id")}
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
	if vr := ValidateHook(dto, enums); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/settings-hooks-one.html", map[string]any{
			"obj":   dto,
			"mods":  worker.List,
			"valid": vr,
		})
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.store.SaveHook(dto); err != nil {
		Err(w, r, err)
		return
	}

	// reload hooks
	LoadHooks(ctrl.store)

	http.Redirect(w, r, "/settings/hooks/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) DeleteHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/hooks/%s?confirm=yes", id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	err := ctrl.store.DeleteHook(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	// reload hooks
	LoadHooks(ctrl.store)
	http.Redirect(w, r, "/settings/hooks/", http.StatusSeeOther)
}
