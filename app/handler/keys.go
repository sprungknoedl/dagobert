package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type KeyCtrl struct {
	store *model.Store
	acl   *ACL

	jobctrl *JobCtrl
}

func NewKeyCtrl(store *model.Store, acl *ACL, jobctrl *JobCtrl) *KeyCtrl {
	return &KeyCtrl{store, acl, jobctrl}
}

func (ctrl KeyCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.ListKeys()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "app/views/keys-many.html", map[string]any{
		"title":   "API Keys",
		"keys":    list,
		"workers": ctrl.jobctrl.Workers(),
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

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "app/views/keys-one.html", map[string]any{
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

	enums, err := ctrl.store.ListEnums()
	if err != nil {
		Err(w, r, err)
		return
	}

	if vr := ValidateKey(dto, enums); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "app/views/keys-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.Key = fp.If(dto.Key == "new", fp.Random(64), dto.Key)
	if err := ctrl.store.SaveKey(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/api-keys/", http.StatusSeeOther)
}

func (ctrl KeyCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/api-keys/%s?confirm=yes", key)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "app/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	if err := ctrl.store.DeleteKey(key); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/api-keys/", http.StatusSeeOther)
}
