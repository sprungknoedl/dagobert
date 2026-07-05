package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/internal/model"
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

	apiKey, apiHash, _ := model.GenerateKey()
	donaldKey, donaldHash, _ := model.GenerateKey()
	bogusKey, bogusHash, _ := model.GenerateKey()

	assert.NoError(t, db.SaveKey(model.Key{Key: apiHash, Name: "admin", Type: "API"}))
	assert.NoError(t, db.SaveKey(model.Key{Key: donaldHash, Name: "triage", Type: "Donald"}))
	assert.NoError(t, db.SaveKey(model.Key{Key: bogusHash, Name: "broken", Type: "Bogus"}))

	acl := NewACL(db)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := ApiKeyMiddleware(db)(acl.Protect(ok))

	do := func(header, key, method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(header, key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("API key reaches an arbitrary path", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(HeaderApiKey, apiKey, "GET", "/cases/"))
	})

	t.Run("API key via Authorization Bearer authenticates identically", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do("Authorization", "Bearer "+apiKey, "GET", "/cases/"))
	})

	t.Run("Donald key may create triage evidence", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(HeaderApiKey, donaldKey, "POST", "/cases/abc/evidences/new"))
	})

	t.Run("Donald key is denied reading cases", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, do(HeaderApiKey, donaldKey, "GET", "/cases/"))
	})

	t.Run("Donald key is denied deleting evidence", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, do(HeaderApiKey, donaldKey, "DELETE", "/cases/abc/evidences/xyz"))
	})

	t.Run("Unknown key type is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, do(HeaderApiKey, bogusKey, "POST", "/cases/abc/evidences/new"))
	})

	t.Run("X-API-Key with a bad checksum is rejected offline", func(t *testing.T) {
		bad := []byte(apiKey)
		bad[len(bad)-1] ^= 1
		assert.Equal(t, http.StatusUnauthorized, do(HeaderApiKey, string(bad), "GET", "/cases/"))
	})

	t.Run("X-API-Key with a wrong prefix is rejected offline", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, do(HeaderApiKey, "xyz_"+apiKey[len(model.KeyPrefix):], "GET", "/cases/"))
	})

	t.Run("X-API-Key with a valid format but unknown hash is rejected", func(t *testing.T) {
		unknown, _, _ := model.GenerateKey()
		assert.Equal(t, http.StatusUnauthorized, do(HeaderApiKey, unknown, "GET", "/cases/"))
	})

	t.Run("non-dgb Authorization Bearer falls through without a principal", func(t *testing.T) {
		var sawUser bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := CurrentUser(r)
			sawUser = err == nil
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest("GET", "/cases/", nil)
		req.Header.Set("Authorization", "Bearer some-other-token")
		rec := httptest.NewRecorder()
		ApiKeyMiddleware(db)(next).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.False(t, sawUser)
	})
}
