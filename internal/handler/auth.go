package handler

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/coreos/go-oidc"
	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
	"golang.org/x/oauth2"
)

var ApiKeyHeader = "X-API-Key"

type OpenIDConfig struct {
	ClientId      string   //id from the authorization service (OIDC provider)
	ClientSecret  string   //secret from the authorization service (OIDC provider)
	ClientUrl     url.URL  //your website's/service's URL for example: "http://localhost:8081/" or "https://mydomain.com/
	Issuer        url.URL  //the URL identifier for the authorization service. for example: "https://accounts.google.com" - try adding "/.well-known/openid-configuration" to the path to make sure it's correct
	Scopes        []string //OAuth scopes. If you're unsure go with: []string{oidc.ScopeOpenID, "profile", "email"}
	Identifier    string   // name of the openid claim used to securely identify a user (e.g. "sub").
	AutoProvision bool
	PostLogoutUrl url.URL //user will be redirected to this URL after he logs out (i.e. accesses the '/logout' endpoint added in 'Init()')
}

type AuthCtrl struct {
	openidConfig  OpenIDConfig
	oauthConfig   *oauth2.Config
	verifier      *oidc.IDTokenVerifier
	autoProvision bool
	store         *model.Store
	acl           *ACL
}

func NewAuthCtrl(store *model.Store, acl *ACL, cfg OpenIDConfig) *AuthCtrl {
	providerCtx := context.Background()
	provider, err := oidc.NewProvider(providerCtx, cfg.Issuer.String())
	if err != nil {
		log.Fatalf("Failed to init OIDC provider: %v \n", err.Error())
	}
	oidcConfig := &oidc.Config{
		ClientID: cfg.ClientId,
	}
	verifier := provider.Verifier(oidcConfig)
	endpoint := provider.Endpoint()
	cfg.ClientUrl.Path = "/auth/callback"
	config := &oauth2.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  cfg.ClientUrl.String(),
		Scopes:       cfg.Scopes,
	}

	return &AuthCtrl{
		openidConfig:  cfg,
		oauthConfig:   config,
		verifier:      verifier,
		autoProvision: cfg.AutoProvision,
		store:         store,
		acl:           acl,
	}
}

func (ctrl AuthCtrl) isAuthorized(r *http.Request) bool {
	sess, _ := SessionStore.Get(r, SessionName)
	claims, _ := sess.Values["oidcClaims"].(map[string]any)

	sub, _ := claims[ctrl.openidConfig.Identifier].(string) // the user that wants to access a resource.
	obj := r.URL.Path                                       // the resource that is going to be accessed.
	act := r.Method                                         // the operation that the user performs on the resource.

	res, err := ctrl.acl.Enforce(sub, obj, act)
	if err != nil {
		log.Printf("|%s| failed to authorize: %v", tty.Yellow("WARN "), err)
	}

	return res
}

func (ctrl AuthCtrl) Protect(next http.Handler) http.Handler {
	gob.Register(map[string]any{})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// api key based authorization
		key := r.Header.Get(ApiKeyHeader)
		if key != "" {
			// TODO: validate api key header
			_, err := ctrl.store.GetKey(key)
			if err != nil {
				Warn(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		// session based authorization
		sess, _ := SessionStore.Get(r, SessionName)
		authenticated := sess.Values["oidcAuthenticated"]
		if authenticated != nil && authenticated.(bool) ||
			strings.HasPrefix(r.URL.Path, "/auth/") ||
			strings.HasPrefix(r.URL.Path, "/dist/") {
			if ctrl.isAuthorized(r) {
				next.ServeHTTP(w, r)
			} else {
				ctrl.Forbidden(w, r)
			}
			return
		}

		state := random(16)
		sess.Values["oidcAuthenticated"] = false
		sess.Values["oidcState"] = state
		sess.Values["oidcOriginalRequestUrl"] = r.URL.String()
		err := sess.Save(r, w)
		if err != nil {
			Err(w, r, err)
			return
		}

		//redirect to authorization server
		http.Redirect(w, r, ctrl.oauthConfig.AuthCodeURL(state), http.StatusFound)
	})
}

func (ctrl AuthCtrl) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionStore.Get(r, SessionName)
	sess.Values["oidcAuthenticated"] = false
	sess.Values["oidcClaims"] = nil
	sess.Values["oidcState"] = nil
	sess.Values["oidcOriginalRequestUrl"] = nil

	err := sess.Save(r, w)
	if err != nil {
		Err(w, r, err)
		return
	}

	logoutUrl := ctrl.openidConfig.Issuer
	logoutUrl.RawQuery = (url.Values{"redirect_uri": []string{ctrl.openidConfig.PostLogoutUrl.String()}}).Encode()
	logoutUrl.Path = "protocol/openid-connect/logout"

	http.Redirect(w, r, logoutUrl.String(), http.StatusFound)
}

func (ctrl AuthCtrl) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := SessionStore.Get(r, SessionName)

	state, ok := (sess.Values["oidcState"]).(string)
	if !ok {
		Warn(w, r, errors.New("get 'state' param didn't match local 'state' value"))
		return
	}

	if r.URL.Query().Get("state") != state {
		Warn(w, r, errors.New("get 'state' param didn't match local 'state' value"))
		return
	}

	oauth2Token, err := ctrl.oauthConfig.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		Warn(w, r, err)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		Warn(w, r, errors.New("no id_token field in oauth2 token"))
		return
	}

	idToken, err := ctrl.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		Warn(w, r, err)
		return
	}

	var claims map[string]interface{}
	err = idToken.Claims(&claims)
	if err != nil {
		Warn(w, r, err)
		return
	}

	originalRequestUrl, ok := (sess.Values["oidcOriginalRequestUrl"]).(string)
	if !ok {
		Warn(w, r, errors.New("failed to parse originalRequestUrl"))
		return
	}

	// check if user exists
	uid := claims[ctrl.openidConfig.Identifier].(string)
	user, err := ctrl.store.GetUser(uid)
	if err != nil && !ctrl.autoProvision {
		http.Redirect(w, r, "/auth/forbidden", http.StatusTemporaryRedirect)
		return
	}

	// store successful authentication
	sess.Values["oidcAuthenticated"] = true
	sess.Values["oidcState"] = nil
	sess.Values["oidcOriginalRequestUrl"] = nil
	sess.Values["oidcClaims"] = claims

	err = sess.Save(r, w)
	if err != nil {
		Err(w, r, err)
		return
	}

	// update fields from identity provider
	user.ID = uid
	user.Role = fp.If(user.Role != "", user.Role, "Read-Only")
	user.Name = claims["name"].(string)
	user.UPN = claims["preferred_username"].(string)
	user.Email = claims["email"].(string)
	user.LastLogin = model.Time(time.Now())
	err = ctrl.store.SaveUser(user)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, originalRequestUrl, http.StatusFound)
}

func (ctrl AuthCtrl) Forbidden(w http.ResponseWriter, r *http.Request) {
	Render(ctrl.store, ctrl.acl, w, r, http.StatusForbidden, "internal/views/auth-forbidden.html", map[string]any{})
}

type ACL struct {
	db       *model.Store
	enforcer *casbin.Enforcer
}

func NewACL(db *model.Store) *ACL {
	enforcer, err := casbin.NewEnforcer("files/model.conf", db)
	if err != nil {
		log.Fatalf("Failed to init Casbin enforcer: %v \n", err.Error())
	}

	enforcer.EnableAutoSave(true)
	return &ACL{db, enforcer}
}

// Enforce decides whether a "subject" can access a "object" with the operation "action", input parameters are usually: (sub, obj, act).
func (acl *ACL) Enforce(rvals ...interface{}) (bool, error) {
	return acl.enforcer.Enforce(rvals...)
}

func (acl *ACL) Allowed(uid string, url string, method string) bool {
	ok, _ := acl.Enforce(uid, url, method)
	return ok
}

func (acl *ACL) DeleteUser(uid string) error {
	if _, err := acl.enforcer.DeleteRolesForUser(uid); err != nil {
		return err
	}
	if _, err := acl.enforcer.DeletePermissionForUser(uid); err != nil {
		return err
	}
	if _, err := acl.enforcer.DeleteUser(uid); err != nil {
		return err
	}

	return nil
}

func (acl *ACL) SaveUserRole(uid string, role string) error {
	_, err := acl.enforcer.DeleteRolesForUser(uid)
	if err != nil {
		return err
	}

	_, err = acl.enforcer.AddRoleForUser(uid, "role::"+role)
	if err != nil {
		return err
	}

	return nil
}

func (acl *ACL) SaveUserPermissions(uid string, role string, cases []string) error {
	_, err := acl.enforcer.DeletePermissionsForUser(uid)
	if err != nil {
		return err
	}

	for _, c := range cases {
		obj := fmt.Sprintf("/cases/%s/*", c)
		act := fp.If(role == "Read-Only", http.MethodGet, "*")
		_, err := acl.enforcer.AddPermissionForUser(uid, obj, act)
		if err != nil {
			return err
		}
	}

	return nil
}

func (acl *ACL) SaveCasePermissions(cid string, users []string) error {
	obj := fmt.Sprintf("/cases/%s/*", cid)
	if err := acl.db.RemoveFilteredPolicy("p", "p", 1, obj); err != nil {
		return err
	}

	for _, uid := range users {
		user, err := acl.db.GetUser(uid)
		if err != nil {
			return err
		}

		act := fp.If(user.Role == "Read-Only", http.MethodGet, "*")
		if err := acl.db.AddPolicy("p", "p", []string{user.ID, obj, act}); err != nil {
			return err
		}
	}

	if err := acl.enforcer.LoadPolicy(); err != nil {
		return err
	}

	return nil
}
