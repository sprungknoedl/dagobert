package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type KeyCtrl struct {
	Ctrl
	jobctrl *JobCtrl
}

func NewKeyCtrl(store *model.Store, acl *ACL, jobctrl *JobCtrl) *KeyCtrl {
	return &KeyCtrl{Ctrl: BaseCtrl{store, acl}, jobctrl: jobctrl}
}

func (ctrl KeyCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.Store().ListKeys()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsKeyMany(Env(ctrl, r), list, ctrl.jobctrl.Workers()))
}

func (ctrl KeyCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	obj := model.Key{Key: key}
	if key != "new" {
		var err error
		obj, err = ctrl.Store().GetKey(key)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsKeysOne(Env(ctrl, r), obj, valid.ValidationError{}))
}

func (ctrl KeyCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Key{}
	err := Decode(ctrl.Store(), r, &dto, ValidateKey)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsKeysOne(Env(ctrl, r), dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	dto.Key = fp.If(dto.Key == "new", fp.Random(64), dto.Key)
	if err := ctrl.Store().SaveKey(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/api-keys/", http.StatusSeeOther)
}

func (ctrl KeyCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/api-keys/%s?confirm=yes", key)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := ctrl.Store().DeleteKey(key); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/api-keys/", http.StatusSeeOther)
}
