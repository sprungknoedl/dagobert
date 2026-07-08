package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) UserList(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListUsers()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsUsersMany(h.Env(r), list), nil)
}

func (h *Handler) UserEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.User{ID: id}
	if id != "new" {
		var err error
		obj, err = h.Store.GetUser(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsUsersOne(h.Env(r), obj, valid.ValidationError{}), nil)
}

func (h *Handler) UserSave(w http.ResponseWriter, r *http.Request) {
	dto := model.User{}
	err := Decode(h.Store, r, &dto, ValidateUser)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsUsersOne(h.Env(r), dto, vr), nil)
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	usr, err := h.Store.GetUser(dto.ID)
	if err != nil {
		Err(w, r, err)
		return
	}

	// Update fields
	usr.ID = dto.ID
	usr.Role = dto.Role
	usr.Name = dto.Name
	usr.UPN = dto.UPN
	usr.Email = dto.Email

	// Update user
	if err := h.Store.SaveUser(usr); err != nil {
		Err(w, r, err)
		return
	}

	// Update role in casbin
	if err := h.ACL.SaveUserRole(usr.ID, usr.Role); err != nil {
		Err(w, r, err)
		return
	}

	// Update permissions casbin, those need to be changed when a role change happens
	perms, err := h.Store.GetUserPermissions(usr.ID)
	if err != nil {
		Err(w, r, err)
		return
	}
	if err := h.ACL.SaveUserPermissions(usr.ID, usr.Role, perms); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/users/", nil)
}

func (h *Handler) UserDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/users/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	if err := h.Store.DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	if err := h.ACL.DeleteUser(id); err != nil {
		Err(w, r, err)
		return
	}

	if GetUser(r).ID == id {
		http.Redirect(w, r, "/auth/logout", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/settings/users/", http.StatusSeeOther)
	}
}

func (h *Handler) UserEditACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := h.Store.GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	cases, err := h.Store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	perms, err := h.Store.GetUserPermissions(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsUsersACL(h.Env(r), obj, cases, perms, valid.ValidationError{}), nil)
}

func (h *Handler) UserSaveACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj, err := h.Store.GetUser(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	form := struct{ Cases []string }{}
	if err := Decode(h.Store, r, &form, nil); err != nil {
		Warn(w, r, err)
		return
	}

	if err := h.ACL.SaveUserPermissions(obj.ID, obj.Role, form.Cases); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/users/", nil)
}
