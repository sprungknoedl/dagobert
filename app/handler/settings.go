package handler

import (
	"github.com/sprungknoedl/dagobert/app/model"
)

type SettingsCtrl struct {
	Ctrl
}

func NewSettingsCtrl(store *model.Store, acl *ACL) *SettingsCtrl {
	return &SettingsCtrl{BaseCtrl{store, acl}}
}
