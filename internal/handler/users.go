package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type UserCtrl struct {
	store *model.Store
}

func NewUserCtrl(store *model.Store) *UserCtrl {
	return &UserCtrl{store: store}
}

func (ctrl UserCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.ListUsers()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/users-many.html", map[string]any{
		"title": "Users",
		"rows":  list,
	})
}

func (ctrl UserCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.User{ID: id}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetUser(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/users-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl UserCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.User{}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	if vr := ValidateUser(dto); !vr.Valid() {
		Render(ctrl.store, w, r, http.StatusUnprocessableEntity, "internal/views/users-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	if err := ctrl.store.SaveUser(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/users/", http.StatusSeeOther)
}

func (ctrl UserCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/users/%s?confirm=yes", id)
		Render(ctrl.store, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	if err := ctrl.store.DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	if GetEnv(ctrl.store, r).Username == id {
		http.Redirect(w, r, "/auth/logout", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
