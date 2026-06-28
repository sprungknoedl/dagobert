package handler

import (
	"net/http"

	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
)

type SettingsCtrl struct {
	Ctrl
}

func NewSettingsCtrl(store *model.Store, acl *auth.ACL) *SettingsCtrl {
	return &SettingsCtrl{BaseCtrl{store, acl}}
}

func (ctrl SettingsCtrl) Overview(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.SettingsOverview(Env(ctrl, r)))
}
