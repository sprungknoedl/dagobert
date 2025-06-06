package handler

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (ctrl SettingsCtrl) ListEnums(w http.ResponseWriter, r *http.Request) {
	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-enums-many.html", map[string]any{
		"title": "Case Objects",
	})
}

func (ctrl SettingsCtrl) EditEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.Enum{ID: id}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetEnum(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/settings-enums-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl SettingsCtrl) SaveEnum(w http.ResponseWriter, r *http.Request) {}

func (ctrl SettingsCtrl) DeleteEnum(w http.ResponseWriter, r *http.Request) {}
