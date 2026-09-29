package auth

import (
	"net/http"
	"net/http/httptest"
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

	assert.Nil(t, db.SaveUser(model.User{ID: "u1", Login: "admin", Role: "Administrator"}))
	assert.Nil(t, db.SaveUser(model.User{ID: "u2", Login: "readonly", Role: "Read-Only"}))

	assert.Nil(t, acl.SaveCasePermissions("case1", []string{"u1", "u2"}))
	assert.True(t, acl.Allowed("u1", "/cases/case1/events/", http.MethodPost))
	assert.True(t, acl.Allowed("u2", "/cases/case1/events/", http.MethodGet))
	assert.False(t, acl.Allowed("u2", "/cases/case1/events/", http.MethodPost))

	// re-saving with a narrower user list revokes access for the dropped user
	assert.Nil(t, acl.SaveCasePermissions("case1", []string{"u1"}))
	assert.False(t, acl.Allowed("u2", "/cases/case1/events/", http.MethodGet))
}

// TestSyncRole checks that a role written by another ACL on the same database
// (as `dagobert create-user` does) is picked up on a mismatch, and that a
// matching role skips the reload.
func TestSyncRole(t *testing.T) {
	db := setupDB(t)
	server := NewACL(db)

	assert.Nil(t, NewACL(db).SaveUserRole("u1", "Administrator"))
	assert.False(t, server.Allowed("u1", "/settings/users/", http.MethodDelete))

	assert.Nil(t, server.SyncRole("u1", "Administrator"))
	assert.True(t, server.Allowed("u1", "/settings/users/", http.MethodDelete))

	t.Run("matching role does not reload", func(t *testing.T) {
		assert.Nil(t, NewACL(db).SaveUserRole("u2", "Administrator"))
		assert.Nil(t, server.SyncRole("u1", "Administrator"))
		assert.False(t, server.Allowed("u2", "/settings/users/", http.MethodDelete))
	})

	t.Run("user without role does not reload", func(t *testing.T) {
		assert.Nil(t, server.SyncRole("u3", ""))
		assert.False(t, server.Allowed("u2", "/settings/users/", http.MethodDelete))
	})
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

func TestIsNavigation(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		fetchMode string
		accept    string
		want      bool
	}{
		{"GET with Sec-Fetch-Mode navigate", http.MethodGet, "navigate", "", true},
		{"GET with Sec-Fetch-Mode cors", http.MethodGet, "cors", "", false},
		{"GET without Sec-Fetch-Mode but html Accept", http.MethodGet, "", "text/html,*/*", true},
		{"GET without Sec-Fetch-Mode and non-html Accept", http.MethodGet, "", "application/json", false},
		{"POST is never a navigation, even with navigate mode", http.MethodPost, "navigate", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.fetchMode != "" {
				req.Header.Set("Sec-Fetch-Mode", tt.fetchMode)
			}
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			assert.Equal(t, tt.want, isNavigation(req))
		})
	}
}

func TestIsPagePath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /cases/", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /reports/{id}/delete", func(w http.ResponseWriter, r *http.Request) {})
	a := &Auth{routes: mux}

	tests := []struct {
		name string
		dst  string
		want bool
	}{
		{"registered GET route", "/cases/", true},
		{"unregistered path", "/nonexistent", false},
		{"route exists but only for a different method", "/reports/1/delete", false},
		{"protocol-relative open redirect rejected", "//evil.example.com", false},
		{"non-relative destination rejected", "https://evil.example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.isPagePath(tt.dst))
		})
	}

	t.Run("nil routes accepts any safe relative path", func(t *testing.T) {
		a := &Auth{}
		assert.True(t, a.isPagePath("/anything"))
	})
}
