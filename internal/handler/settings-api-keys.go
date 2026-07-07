package handler

import (
	"fmt"
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) APIKeyList(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListAPIKeys()
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.SettingsAPIKeysMany(h.Env(r), list))
}

func (h *Handler) APIKeyEdit(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	obj := model.APIKey{Key: key}
	if key != "new" {
		var err error
		obj, err = h.Store.GetAPIKey(key)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.SettingsAPIKeysOne(h.Env(r), obj, valid.ValidationError{}))
}

func (h *Handler) APIKeySave(w http.ResponseWriter, r *http.Request) {
	dto := model.APIKey{}
	err := Decode(h.Store, r, &dto, ValidateAPIKey)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.SettingsAPIKeysOne(h.Env(r), dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	if dto.Key == "new" {
		plaintext, hash, hint := model.GenerateAPIKey()
		if err := h.Store.SaveAPIKey(model.APIKey{Key: hash, Hint: hint, Type: dto.Type, Name: dto.Name}); err != nil {
			Err(w, r, err)
			return
		}

		// reveal the plaintext exactly once; it lives only in this response
		Render(w, r, http.StatusOK, views.SettingsAPIKeysReveal(h.Env(r), plaintext))
		return
	}

	// existing key: load the stored row and update only Type/Name so the
	// persisted hash/Hint are preserved (don't trust client-supplied values)
	obj, err := h.Store.GetAPIKey(dto.Key)
	if err != nil {
		Err(w, r, err)
		return
	}
	obj.Type = dto.Type
	obj.Name = dto.Name
	if err := h.Store.SaveAPIKey(obj); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, "/settings/api-keys/")
}

func (h *Handler) APIKeyDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/settings/api-keys/%s?confirm=yes", key)
		Render(w, r, http.StatusOK, views.ConfirmDialog(uri))
		return
	}

	if err := h.Store.DeleteAPIKey(key); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/settings/api-keys/", http.StatusSeeOther)
}
