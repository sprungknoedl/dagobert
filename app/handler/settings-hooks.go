package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/app/worker"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListHooks(w http.ResponseWriter, r *http.Request) {
	hooks, err := ctrl.Store().ListHooks()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsHooksMany(Env(ctrl, r), hooks))
}

func (ctrl SettingsCtrl) EditHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.Hook{ID: id}
	if id != "new" {
		var err error
		obj, err = ctrl.Store().GetHook(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsHooksOne(Env(ctrl, r), obj, worker.List, valid.Result{}))
}

func (ctrl SettingsCtrl) SaveHook(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.Hook{ID: r.PathValue("id")}
	if err := Decode(r, &dto); err != nil {
		Err(w, r, err)
		return
	}

	enums, err := ctrl.Store().ListEnums()
	if err != nil {
		Err(w, r, err)
		return
	}

	// validate form
	if vr := ValidateHook(dto, enums); !vr.Valid() {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsHooksOne(Env(ctrl, r), dto, worker.List, vr))
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveHook(dto); err != nil {
		Err(w, r, err)
		return
	}

	// reload hooks
	LoadHooks(ctrl.Store())

	http.Redirect(w, r, "/settings/hooks/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) DeleteHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/hooks/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := ctrl.Store().DeleteHook(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	// reload hooks
	LoadHooks(ctrl.Store())
	http.Redirect(w, r, "/settings/hooks/", http.StatusSeeOther)
}
