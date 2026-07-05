package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) KeyList(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListKeys()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsKeyMany(h.Env(r), list))
}

func (h *Handler) KeyEdit(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	obj := model.Key{Key: key}
	if key != "new" {
		var err error
		obj, err = h.Store.GetKey(key)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsKeysOne(h.Env(r), obj, valid.ValidationError{}))
}

func (h *Handler) KeySave(w http.ResponseWriter, r *http.Request) {
	dto := model.Key{}
	err := Decode(h.Store, r, &dto, ValidateKey)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsKeysOne(h.Env(r), dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	if dto.Key == "new" {
		plaintext, hash, hint := model.GenerateKey()
		if err := h.Store.SaveKey(model.Key{Key: hash, Hint: hint, Type: dto.Type, Name: dto.Name}); err != nil {
			Err(w, r, err)
			return
		}

		// reveal the plaintext exactly once; it lives only in this response
		Render(w, r, http.StatusOK, views.SettingsKeyReveal(h.Env(r), plaintext))
		return
	}

	// existing key: load the stored row and update only Type/Name so the
	// persisted hash/Hint are preserved (don't trust client-supplied values)
	obj, err := h.Store.GetKey(dto.Key)
	if err != nil {
		Err(w, r, err)
		return
	}
	obj.Type = dto.Type
	obj.Name = dto.Name
	if err := h.Store.SaveKey(obj); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/api-keys/")
}

func (h *Handler) KeyDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/api-keys/%s?confirm=yes", key)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := h.Store.DeleteKey(key); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/api-keys/", http.StatusSeeOther)
}
