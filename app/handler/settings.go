package handler

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/app/views"
)

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsOverview(h.Env(r)))
}
