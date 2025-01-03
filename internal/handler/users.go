package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type UserCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewUserCtrl(store *model.Store, acl *ACL) *UserCtrl {
	return &UserCtrl{store, acl}
}

func (ctrl UserCtrl) List(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.ListUsers()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/users-many.html", map[string]any{
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

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/users-one.html", map[string]any{
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
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/users-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	// Update user
	if err := ctrl.store.SaveUser(dto); err != nil {
		Err(w, r, err)
		return
	}

	// Update role in casbin
	if err := ctrl.acl.SaveUserRole(dto.ID, dto.Role); err != nil {
		Err(w, r, err)
		return
	}

	// Update permissions casbin, those need to be changed when a role change happens
	perms, err := ctrl.store.GetUserPermissions(dto.ID)
	if err != nil {
		Err(w, r, err)
		return
	}
	if err := ctrl.acl.SaveUserPermissions(dto.ID, dto.Role, perms); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "user:"+dto.ID, "Saved user: %s (%s)", dto.Name, dto.UPN)
	http.Redirect(w, r, "/settings/users/", http.StatusSeeOther)
}

func (ctrl UserCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/users/%s?confirm=yes", id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	obj, err := ctrl.store.GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	if err := ctrl.store.DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	if err := ctrl.acl.DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "user:"+obj.ID, "Deleted user: %s (%s)", obj.Name, obj.UPN)
	if GetEnv(ctrl.store, r).UID == id {
		http.Redirect(w, r, "/auth/logout", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (ctrl UserCtrl) EditACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := ctrl.store.GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	cases, err := ctrl.store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	perms, err := ctrl.store.GetUserPermissions(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/users-acl.html", map[string]any{
		"obj":   obj,
		"perms": perms,
		"cases": cases,
		"valid": valid.Result{},
	})
}

func (ctrl UserCtrl) SaveACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := ctrl.store.GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	form := struct{ Cases []string }{}
	if err := Decode(r, &form); err != nil {
		Warn(w, r, err)
		return
	}

	if err := ctrl.acl.SaveUserPermissions(obj.ID, obj.Role, form.Cases); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "user:"+obj.ID, "Allowed %s (%s) access to %v", obj.Name, obj.UPN, form.Cases)
	http.Redirect(w, r, "/settings/users/", http.StatusSeeOther)
}
