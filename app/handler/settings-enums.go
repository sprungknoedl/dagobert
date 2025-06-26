package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListEnums(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsEnumsMany(Env(ctrl, r)))
}

func (ctrl SettingsCtrl) EditEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cat := r.URL.Query().Get("category")
	obj, err := GetObject(id, model.Enum{ID: id, Category: cat}, ctrl.Store().GetEnum)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsEnumsOne(Env(ctrl, r), obj, valid.Result{}))
}

func (ctrl SettingsCtrl) SaveEnum(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.Enum{ID: r.PathValue("id")}
	if err := Decode(r, &dto); err != nil {
		Err(w, r, err)
		return
	}

	// validate form
	if vr := ValidateEnum(dto); !vr.Valid() {
		slog.Info("invalid enum", "valid", vr)

		Render(w, r, http.StatusUnprocessableEntity, views.SettingsEnumsOne(Env(ctrl, r), dto, vr))
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveEnum(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/enums/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) DeleteEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/enums/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := ctrl.Store().DeleteEnum(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/enums/", http.StatusSeeOther)
}
