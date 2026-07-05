package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/internal/worker"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) ListHooks(w http.ResponseWriter, r *http.Request) {
	hooks, err := h.Store.ListHooks()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsHooksMany(h.Env(r), hooks))
}

func (h *Handler) EditHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	obj := model.Hook{ID: id}
	if id != "new" {
		var err error
		obj, err = h.Store.GetHook(id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	enum := fp.ToList(fp.ApplyM(worker.Modules, func(m model.Module) model.Enum { return model.Enum{Name: m.Name()} }))
	Render(w, r, http.StatusOK, views.SettingsHooksOne(h.Env(r), obj, enum, valid.ValidationError{}))
}

func (h *Handler) SaveHook(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.Hook{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateHook)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		enum := fp.ToList(fp.ApplyM(worker.Modules, func(m model.Module) model.Enum { return model.Enum{Name: m.Name()} }))
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsHooksOne(h.Env(r), dto, enum, vr))
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveHook(dto); err != nil {
		Err(w, r, err)
		return
	}

	// reload hooks
	worker.LoadHooks(h.Store)

	RedirectAfterSave(w, r, "/settings/hooks/")
}

func (h *Handler) DeleteHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/hooks/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := h.Store.DeleteHook(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	// reload hooks
	worker.LoadHooks(h.Store)
	http.Redirect(w, r, "/settings/hooks/", http.StatusSeeOther)
}
