package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) ListEnums(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsEnumsMany(h.Env(r)))
}

func (h *Handler) EditEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cat := r.URL.Query().Get("category")
	obj, err := GetObject(id, model.Enum{ID: id, Category: cat}, h.Store.GetEnum)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsEnumsOne(h.Env(r), obj, valid.ValidationError{}))
}

func (h *Handler) SaveEnum(w http.ResponseWriter, r *http.Request) {
	// deocde form
	dto := model.Enum{ID: r.PathValue("id")}
	err := Decode(h.Store, r, &dto, ValidateEnum)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsEnumsOne(h.Env(r), dto, vr))
		return
	} else if err != nil {
		Err(w, r, err)
		return
	}

	// save database object
	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveEnum(dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/enums/")
}

func (h *Handler) DeleteEnum(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/enums/%s?confirm=yes", id)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	err := h.Store.DeleteEnum(id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/enums/", http.StatusSeeOther)
}
