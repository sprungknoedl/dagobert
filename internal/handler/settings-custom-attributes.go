package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) CustomAttributeList(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsCustomAttributesMany(h.Env(r)), nil)
}

func (h *Handler) CustomAttributeEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entity := r.URL.Query().Get("entity")
	obj, err := GetObject(id, model.CustomAttribute{ID: id, Entity: entity}, h.Store.GetCustomAttribute)
	if errors.Is(err, model.ErrNotFound) {
		NotFound(w, r, err)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsCustomAttributesOne(h.Env(r), obj, valid.ValidationError{}), nil)
}

func (h *Handler) CustomAttributeSave(w http.ResponseWriter, r *http.Request) {
	dto := model.CustomAttribute{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateCustomAttribute)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsCustomAttributesOne(h.Env(r), dto, vr), nil)
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// normalise the comma-separated options input (trim spaces, drop empties)
	opts := fp.Apply(strings.Split(r.FormValue("Options"), ","), strings.TrimSpace)
	dto.Options = model.Strings(fp.Filter(opts, func(s string) bool { return s != "" }))

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveCustomAttribute(dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/custom-attributes/", nil)
}

func (h *Handler) CustomAttributeDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/custom-attributes/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	err := h.Store.DeleteCustomAttribute(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/custom-attributes/", http.StatusSeeOther)
}
