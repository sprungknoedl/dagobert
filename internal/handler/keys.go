package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
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
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, "internal/views/keys-many.html", map[string]any{
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
			utils.Err(w, r, err)
			return
		}
	}

	utils.Render(ctrl.store, w, r, "internal/views/keys-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl KeyCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Key{}
	if err := utils.Decode(r, &dto); err != nil {
		utils.Warn(w, r, err)
		return
	}

	if vr := ValidateKey(dto); !vr.Valid() {
		utils.Render(ctrl.store, w, r, "internal/views/keys-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.Key = utils.If(dto.Key == "new", "", dto.Key)
	if err := ctrl.store.SaveKey(dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}

func (ctrl KeyCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/api-keys/%s?confirm=yes", key)
		utils.Render(ctrl.store, w, r, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
	}

	if err := ctrl.store.DeleteKey(key); err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}
