package auth

import (
	"net/http"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSaveUserRole(t *testing.T) {
	db := setupDB(t)
	acl := NewACL(db)
	uid := "u1"

	assert.Nil(t, acl.SaveUserRole(uid, "Administrator"))
	assert.True(t, acl.Allowed(uid, "/settings/users/", http.MethodDelete))

	// re-assigning a role replaces the prior grant rather than accumulating it
	assert.Nil(t, acl.SaveUserRole(uid, "Read-Only"))
	assert.True(t, acl.Allowed(uid, "/", http.MethodGet))
	assert.False(t, acl.Allowed(uid, "/settings/users/", http.MethodDelete))
}

func TestSaveUserPermissions(t *testing.T) {
	db := setupDB(t)
	acl := NewACL(db)
	uid := "u1"

	assert.Nil(t, acl.SaveUserPermissions(uid, "User", []string{"case1"}))
	assert.True(t, acl.Allowed(uid, "/cases/case1/events/", http.MethodPost))
	assert.False(t, acl.Allowed(uid, "/cases/case2/events/", http.MethodPost))

	t.Run("Read-Only role is gated to GET", func(t *testing.T) {
		assert.Nil(t, acl.SaveUserPermissions(uid, "Read-Only", []string{"case1"}))
		assert.True(t, acl.Allowed(uid, "/cases/case1/events/", http.MethodGet))
		assert.False(t, acl.Allowed(uid, "/cases/case1/events/", http.MethodPost))
	})
}

func TestSaveCasePermissions(t *testing.T) {
	db := setupDB(t)
	acl := NewACL(db)

	assert.Nil(t, db.SaveUser(model.User{ID: "u1", UPN: "admin", Role: "Administrator"}))
	assert.Nil(t, db.SaveUser(model.User{ID: "u2", UPN: "readonly", Role: "Read-Only"}))

	assert.Nil(t, acl.SaveCasePermissions("case1", []string{"u1", "u2"}))
	assert.True(t, acl.Allowed("u1", "/cases/case1/events/", http.MethodPost))
	assert.True(t, acl.Allowed("u2", "/cases/case1/events/", http.MethodGet))
	assert.False(t, acl.Allowed("u2", "/cases/case1/events/", http.MethodPost))

	// re-saving with a narrower user list revokes access for the dropped user
	assert.Nil(t, acl.SaveCasePermissions("case1", []string{"u1"}))
	assert.False(t, acl.Allowed("u2", "/cases/case1/events/", http.MethodGet))
}

func TestDeleteUser(t *testing.T) {
	db := setupDB(t)
	acl := NewACL(db)
	uid := "u1"

	assert.Nil(t, acl.SaveUserRole(uid, "Administrator"))
	assert.True(t, acl.Allowed(uid, "/settings/users/", http.MethodDelete))

	assert.Nil(t, acl.DeleteUser(uid))
	assert.False(t, acl.Allowed(uid, "/settings/users/", http.MethodDelete))
}
