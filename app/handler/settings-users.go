package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type UserCtrl struct {
	Ctrl
}

func NewUserCtrl(store *model.Store, acl *ACL) *UserCtrl {
	return &UserCtrl{BaseCtrl{store, acl}}
}

func (ctrl UserCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.Store().ListUsers()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsUsersMany(Env(ctrl, r), list))
}

func (ctrl UserCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.User{ID: id}
	if id != "new" {
		var err error
		obj, err = ctrl.Store().GetUser(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsUsersOne(Env(ctrl, r), obj, valid.ValidationError{}))
}

func (ctrl UserCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.User{}
	err := Decode(ctrl.Store(), r, &dto, ValidateUser)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsUsersOne(Env(ctrl, r), dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// Update user
	if err := ctrl.Store().SaveUser(dto); err != nil {
		Err(w, r, err)
		return
	}

	// Update role in casbin
	if err := ctrl.ACL().SaveUserRole(dto.ID, dto.Role); err != nil {
		Err(w, r, err)
		return
	}

	// Update permissions casbin, those need to be changed when a role change happens
	perms, err := ctrl.Store().GetUserPermissions(dto.ID)
	if err != nil {
		Err(w, r, err)
		return
	}
	if err := ctrl.ACL().SaveUserPermissions(dto.ID, dto.Role, perms); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/users/", http.StatusSeeOther)
}

func (ctrl UserCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/users/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := ctrl.Store().DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	if err := ctrl.ACL().DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	if GetUser(ctrl.Store(), r).ID == id {
		http.Redirect(w, r, "/auth/logout", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (ctrl UserCtrl) EditACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := ctrl.Store().GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	cases, err := ctrl.Store().ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	perms, err := ctrl.Store().GetUserPermissions(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsUsersACL(Env(ctrl, r), obj, cases, perms, valid.ValidationError{}))
}

func (ctrl UserCtrl) SaveACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := ctrl.Store().GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	form := struct{ Cases []string }{}
	if err := Decode(ctrl.Store(), r, &form, nil); err != nil {
		Warn(w, r, err)
		return
	}

	if err := ctrl.ACL().SaveUserPermissions(obj.ID, obj.Role, form.Cases); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/users/", http.StatusSeeOther)
}
