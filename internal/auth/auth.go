// Package auth handles session-based authentication: OIDC login/logout, API keys, and password changes.
package auth

import (
	"cmp"
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	SessionUserID = "user_id"
	SessionState  = "oauth_state"
	SessionRedir  = "redir" // post-login destination
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
)

type Auth struct {
	store    *model.Store
	session  *scs.SessionManager
	provider *oidc.Provider
	oauth2   oauth2.Config
	routes   *http.ServeMux // secured mux, used to validate post-login redirects
}

// SetRoutes registers the secured mux so redirectAfterLogin can verify that a
// stored destination resolves to a real GET handler before redirecting to it.
func (a *Auth) SetRoutes(mux *http.ServeMux) { a.routes = mux }

func New(store *model.Store, session *scs.SessionManager) (*Auth, error) {
	a := &Auth{store: store, session: session}
	if os.Getenv("OIDC_ENABLED") == "true" {
		p, err := oidc.NewProvider(context.Background(), os.Getenv("OIDC_ISSUER"))
		if err != nil {
			return nil, err
		}
		a.provider = p
		a.oauth2 = oauth2.Config{
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			// Derived from the existing env var; no new OIDC_REDIRECT_URL.
			RedirectURL: strings.TrimSuffix(os.Getenv("OIDC_CLIENT_URL"), "/") + "/auth/callback",
			Endpoint:    p.Endpoint(),
			Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}
	return a, nil
}

// LoadUser resolves the current user once per request and stores it in the
// request context. ApiKeyMiddleware may already have placed the system user
// there; in that case the session is not consulted.
func (a *Auth) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(ctxKeyUser).(*model.User); ok {
			next.ServeHTTP(w, r)
			return
		}
		if id := a.session.GetString(r.Context(), SessionUserID); id != "" {
			if user, err := a.store.GetUser(id); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, &user))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func CurrentUser(r *http.Request) (*model.User, error) {
	user, ok := r.Context().Value(ctxKeyUser).(*model.User)
	if !ok {
		return nil, errors.New("not authenticated")
	}
	return user, nil
}

// Require gates the secured mux. It checks the context (set by LoadUser or
// ApiKeyMiddleware), not the session directly.
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := CurrentUser(r); err != nil {
			// Only remember the destination for genuine page navigations.
			// Background subresource fetches (e.g. /favicon.ico) also hit this
			// middleware while unauthenticated and would otherwise clobber the
			// stored redirect, sending the user there after login.
			if isNavigation(r) {
				a.session.Put(r.Context(), SessionRedir, r.URL.Path)
			}
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isNavigation reports whether r is a top-level page navigation rather than a
// background subresource request. It prefers the Sec-Fetch-Mode header (sent by
// all modern browsers) and falls back to the Accept header for clients that
// omit it.
func isNavigation(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if mode := r.Header.Get("Sec-Fetch-Mode"); mode != "" {
		return mode == "navigate"
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// redirectAfterLogin pops the stored post-login destination and redirects there
// if it is a safe relative path, otherwise to the index.
func (a *Auth) redirectAfterLogin(w http.ResponseWriter, r *http.Request) {
	dst := a.session.PopString(r.Context(), SessionRedir)
	if !a.isPagePath(dst) {
		dst = "/"
	}
	http.Redirect(w, r, dst, http.StatusFound)
}

// isPagePath reports whether dst is a safe relative path that resolves to a
// registered GET handler on the secured mux. This rejects open redirects as
// well as destinations that only exist for non-GET methods or aren't routes at
// all.
func (a *Auth) isPagePath(dst string) bool {
	if !strings.HasPrefix(dst, "/") || strings.HasPrefix(dst, "//") {
		return false
	}
	if a.routes == nil {
		return true
	}
	probe := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: dst}}
	_, pattern := a.routes.Handler(probe)
	return pattern != ""
}

// LoginOIDC is the entry point for the "Sign in with SSO" button.
// GET /auth/oidc
func (a *Auth) LoginOIDC(w http.ResponseWriter, r *http.Request) {
	state := fp.Random(32)
	a.session.Put(r.Context(), SessionState, state)
	http.Redirect(w, r, a.oauth2.AuthCodeURL(state), http.StatusFound)
}

// Callback handles the OIDC redirect.
// GET /auth/callback
func (a *Auth) Callback(w http.ResponseWriter, r *http.Request) {
	if s := a.session.PopString(r.Context(), SessionState); s == "" || r.FormValue("state") != s {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	token, err := a.oauth2.Exchange(r.Context(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	raw, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	idToken, err := a.provider.Verifier(&oidc.Config{ClientID: a.oauth2.ClientID}).Verify(r.Context(), raw)
	if err != nil {
		http.Error(w, "id_token verification failed", http.StatusInternalServerError)
		return
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "failed to parse claims", http.StatusInternalServerError)
		return
	}
	str := func(k string) string { v, _ := claims[k].(string); return v }

	// Identity = the configured ID claim (default "oid")
	id := str(cmp.Or(os.Getenv("OIDC_ID_CLAIM"), "oid"))
	if id == "" {
		http.Error(w, "id claim missing from token", http.StatusInternalServerError)
		return
	}

	user, err := a.store.GetUser(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if os.Getenv("OIDC_AUTO_PROVISION") != "true" {
			http.Error(w, "unknown user", http.StatusForbidden)
			return
		}
		// New users are provisioned with no role; Casbin denies everything
		// until an administrator assigns one.
		user = model.User{ID: id}
	} else if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}

	// Refresh profile fields from the token on every login
	user.Name = str("name")
	user.UPN = str("preferred_username")
	user.Email = str("email")
	user.LastLogin = model.Time(time.Now())
	if err := a.store.SaveUser(user); err != nil {
		http.Error(w, "failed to save user", http.StatusInternalServerError)
		return
	}

	a.session.RenewToken(r.Context())
	a.session.Put(r.Context(), SessionUserID, user.ID)
	a.redirectAfterLogin(w, r)
}

// LoginLocal serves the login form and handles local password authentication.
// The form is always served; the SSO button appears next to it when
// OIDC_ENABLED=true (invariant 5).
// GET/POST /auth/login
func (a *Auth) LoginLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		views.Login("").Render(r.Context(), w)
		return
	}

	user, err := a.store.GetUserByUPN(r.FormValue("email")) // UPN, see invariant 2
	if err != nil || user.Password == "" ||
		bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.FormValue("password"))) != nil {
		views.Login("Invalid username or password").Render(r.Context(), w)
		return
	}

	user.LastLogin = model.Time(time.Now())
	a.store.SaveUser(user)

	a.session.RenewToken(r.Context())
	a.session.Put(r.Context(), SessionUserID, user.ID)
	a.redirectAfterLogin(w, r)
}

// Logout destroys the session.
// GET /auth/logout (invariant 7 — nav link is a GET)
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	a.session.Destroy(r.Context())
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}
