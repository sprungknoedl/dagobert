package handler

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/internal/views"
)

func (h *Handler) SettingsOverview(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsOverview(h.Env(r)), nil)
}
