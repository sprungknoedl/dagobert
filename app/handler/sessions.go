package handler

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

// Session is the application-wide session manager. It is initialized by
// InitSession during handler.Run and shared with the auth layer.
var Session *scs.SessionManager

func InitSession(db *sql.DB) {
	Session = scs.New()
	Session.Store = sqlite3store.New(db) // runs its own expired-session cleanup goroutine
	Session.Lifetime = 24 * time.Hour
	Session.Cookie.HttpOnly = true
	Session.Cookie.SameSite = http.SameSiteLaxMode
	// HTTPS-only by default; relax for local development over plain HTTP.
	Session.Cookie.Secure = os.Getenv("WEB_SECURE") != "false"
}
