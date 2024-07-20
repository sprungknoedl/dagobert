package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type KeyCtrl struct {
	store *model.Store
}

func NewKeyCtrl(store *model.Store) *KeyCtrl {
	return &KeyCtrl{store}
}

func (ctrl KeyCtrl) List(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindKeys(search, sort)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/keys-many.html", map[string]any{
		"title": "API Keys",
		"rows":  list,
	})
}

func (ctrl KeyCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	obj := model.Key{Key: key}
	if key != "new" {
		var err error
		obj, err = ctrl.store.GetKey(key)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/keys-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl KeyCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Key{}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	if vr := ValidateKey(dto); !vr.Valid() {
		Render(ctrl.store, w, r, http.StatusUnprocessableEntity, "internal/views/keys-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.Key = fp.If(dto.Key == "new", "", dto.Key)
	if err := ctrl.store.SaveKey(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/settings/api-keys/"), http.StatusSeeOther)
}

func (ctrl KeyCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/api-keys/%s?confirm=yes", key)
		Render(ctrl.store, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	if err := ctrl.store.DeleteKey(key); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/settings/api-keys/"), http.StatusSeeOther)
}
