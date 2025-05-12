package handler

import (
	"github.com/sprungknoedl/dagobert/internal/model"
)

type SettingsCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewSettingsCtrl(store *model.Store, acl *ACL) *SettingsCtrl {
	return &SettingsCtrl{store, acl}
}
