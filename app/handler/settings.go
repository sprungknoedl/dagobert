package handler

import (
	"github.com/sprungknoedl/dagobert/app/model"
)

type SettingsCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewSettingsCtrl(store *model.Store, acl *ACL) *SettingsCtrl {
	return &SettingsCtrl{store, acl}
}
