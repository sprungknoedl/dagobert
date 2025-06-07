package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListEnums(w http.ResponseWriter, r *http.Request) {
	enums, _ := ctrl.store.ListEnums()
	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-enums-many.html", map[string]any{
		"title": "Case Objects",
		"enums": enums,
	})
}

func (ctrl SettingsCtrl) EditEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cat := r.URL.Query().Get("category")
	obj, err := GetObject(id, model.Enum{ID: id, Category: cat}, ctrl.store.GetEnum)
	if err != nil {
		Err(w, r, err)
		return
	}

	// _ = views.EnumFormView(GoEnv(ctrl.store, ctrl.acl, r), obj, valid.Result{}).Render(w)
	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-enums-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
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

		w.WriteHeader(http.StatusUnprocessableEntity)
		// _ = views.EnumFormView(GoEnv(ctrl.store, ctrl.acl, r), dto, vr).Render(w)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-enums-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.store.SaveEnum(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/enums/", http.StatusSeeOther)
}

func (ctrl SettingsCtrl) DeleteEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/enums/%s?confirm=yes", id)
		// _ = views.ConfirmDialog(GoEnv(ctrl.store, ctrl.acl, r), uri).Render(w)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	err := ctrl.store.DeleteEnum(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/enums/", http.StatusSeeOther)
}
