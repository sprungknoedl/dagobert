package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) ValueListList(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsValueListsMany(h.Env(r)), nil)
}

func (h *Handler) ValueListEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cat := r.URL.Query().Get("category")
	obj, err := GetObject(id, model.ValueListItem{ID: id, Category: cat}, h.Store.GetEnum)
	if errors.Is(err, model.ErrNotFound) {
		NotFound(w, r, err)
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsValueListsOne(h.Env(r), obj, valid.ValidationError{}), nil)
}

func (h *Handler) ValueListSave(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.ValueListItem{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateValueListItem)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsValueListsOne(h.Env(r), dto, vr), nil)
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveEnum(dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/value-lists/", nil)
}

func (h *Handler) ValueListDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/value-lists/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri), nil)
		return
	}

	err := h.Store.DeleteEnum(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/value-lists/", http.StatusSeeOther)
}
