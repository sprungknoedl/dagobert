package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/stretchr/testify/assert"
)

func setupDB(t *testing.T) *model.Store {
	db, err := model.Connect(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.RawConn.Close() })

	source, _ := iofs.New(model.Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}

	return db
}

func TestApiKeyMiddleware(t *testing.T) {
	db := setupDB(t)

	assert.NoError(t, db.SaveKey(model.Key{Key: "api-key", Name: "admin", Type: "API"}))
	assert.NoError(t, db.SaveKey(model.Key{Key: "donald-key", Name: "triage", Type: "Donald"}))
	assert.NoError(t, db.SaveKey(model.Key{Key: "bogus-key", Name: "broken", Type: "Bogus"}))

	acl := NewACL(db)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := ApiKeyMiddleware(db)(acl.Protect(ok))

	do := func(key, method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(HeaderApiKey, key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("API key reaches an arbitrary path", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do("api-key", "GET", "/cases/"))
	})

	t.Run("Donald key may create triage evidence", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do("donald-key", "POST", "/cases/abc/evidences/new"))
	})

	t.Run("Donald key is denied reading cases", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, do("donald-key", "GET", "/cases/"))
	})

	t.Run("Donald key is denied deleting evidence", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, do("donald-key", "DELETE", "/cases/abc/evidences/xyz"))
	})

	t.Run("Unknown key type is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, do("bogus-key", "POST", "/cases/abc/evidences/new"))
	})
}
