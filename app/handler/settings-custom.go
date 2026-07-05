package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) ListCustomAttributes(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsCustomMany(h.Env(r)))
}

func (h *Handler) EditCustomAttribute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entity := r.URL.Query().Get("entity")
	obj, err := GetObject(id, model.CustomAttribute{ID: id, Entity: entity}, h.Store.GetCustomAttribute)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsCustomOne(h.Env(r), obj, valid.ValidationError{}))
}

func (h *Handler) SaveCustomAttribute(w http.ResponseWriter, r *http.Request) {
	dto := model.CustomAttribute{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateCustomAttribute)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsCustomOne(h.Env(r), dto, vr))
		return
	} else if err != nil {
		Err(w, r, err)
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

	RedirectAfterSave(w, r, "/settings/custom/")
}

func (h *Handler) DeleteCustomAttribute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/custom/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := h.Store.DeleteCustomAttribute(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/custom/", http.StatusSeeOther)
}
