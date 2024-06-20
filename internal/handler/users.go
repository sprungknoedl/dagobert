package handler

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/gorilla/sessions"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
	"golang.org/x/oauth2"
)

var SessionName = "default"
var SessionStore = sessions.NewCookieStore([]byte(os.Getenv("WEB_SESSION_SECRET")))
var ApiKeyHeader = "X-API-Key"

type OpenIDConfig struct {
	ClientId      string   //id from the authorization service (OIDC provider)
	ClientSecret  string   //secret from the authorization service (OIDC provider)
	ClientUrl     url.URL  //your website's/service's URL for example: "http://localhost:8081/" or "https://mydomain.com/
	Issuer        url.URL  //the URL identifier for the authorization service. for example: "https://accounts.google.com" - try adding "/.well-known/openid-configuration" to the path to make sure it's correct
	Scopes        []string //OAuth scopes. If you're unsure go with: []string{oidc.ScopeOpenID, "profile", "email"}
	Identifier    string   // name of the openid claim used to securely identify a user (e.g. "sub").
	PostLogoutUrl url.URL  //user will be redirected to this URL after he logs out (i.e. accesses the '/logout' endpoint added in 'Init()')
}

type UserCtrl struct {
	openidConfig OpenIDConfig
	oauthConfig  *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	store        *model.Store
}

func NewUserCtrl(store *model.Store, cfg OpenIDConfig) *UserCtrl {
	providerCtx := context.Background()
	provider, err := oidc.NewProvider(providerCtx, cfg.Issuer.String())
	if err != nil {
		log.Fatalf("Failed to init OIDC provider. Error: %v \n", err.Error())
	}
	oidcConfig := &oidc.Config{
		ClientID: cfg.ClientId,
	}
	verifier := provider.Verifier(oidcConfig)
	endpoint := provider.Endpoint()
	cfg.ClientUrl.Path = "oidc-callback"
	config := &oauth2.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  cfg.ClientUrl.String(),
		Scopes:       cfg.Scopes,
	}

	return &UserCtrl{
		openidConfig: cfg,
		oauthConfig:  config,
		verifier:     verifier,
		store:        store,
	}
}

func (ctrl UserCtrl) Protect(next http.Handler) http.Handler {
	gob.Register(map[string]any{})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// api key based authorization
		key := r.Header.Get(ApiKeyHeader)
		if key != "" {
			// TODO: validate api key header
			_, err := ctrl.store.GetKey(key)
			if err != nil {
				utils.Warn(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		// session based authorization
		sess, _ := SessionStore.Get(r, SessionName)
		authorized := sess.Values["oidcAuthorized"]
		if (authorized != nil && authorized.(bool)) || r.URL.Path == "/oidc-callback" {
			next.ServeHTTP(w, r)
			return
		}

		state := random(16)
		sess.Values["oidcAuthorized"] = false
		sess.Values["oidcState"] = state
		sess.Values["oidcOriginalRequestUrl"] = r.URL.String()
		err := sess.Save(r, w)
		if err != nil {
			utils.Err(w, r, err)
			return
		}

		//redirect to authorization server
		http.Redirect(w, r, ctrl.oauthConfig.AuthCodeURL(state), http.StatusFound)
	})
}

func (ctrl UserCtrl) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionStore.Get(r, SessionName)
	sess.Values["oidcAuthorized"] = false
	sess.Values["oidcClaims"] = nil
	sess.Values["oidcState"] = nil
	sess.Values["oidcOriginalRequestUrl"] = nil

	err := sess.Save(r, w)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	logoutUrl := ctrl.openidConfig.Issuer
	logoutUrl.RawQuery = (url.Values{"redirect_uri": []string{ctrl.openidConfig.PostLogoutUrl.String()}}).Encode()
	logoutUrl.Path = "protocol/openid-connect/logout"

	http.Redirect(w, r, logoutUrl.String(), http.StatusFound)
}

func (ctrl UserCtrl) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, _ := SessionStore.Get(r, SessionName)

	state, ok := (sess.Values["oidcState"]).(string)
	if !ok {
		utils.Warn(w, r, errors.New("get 'state' param didn't match local 'state' value"))
		return
	}

	if r.URL.Query().Get("state") != state {
		utils.Warn(w, r, errors.New("get 'state' param didn't match local 'state' value"))
		return
	}

	oauth2Token, err := ctrl.oauthConfig.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		utils.Warn(w, r, err)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		utils.Warn(w, r, errors.New("no id_token field in oauth2 token"))
		return
	}

	idToken, err := ctrl.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		utils.Warn(w, r, err)
		return
	}

	var claims map[string]interface{}
	err = idToken.Claims(&claims)
	if err != nil {
		utils.Warn(w, r, err)
		return
	}

	originalRequestUrl, ok := (sess.Values["oidcOriginalRequestUrl"]).(string)
	if !ok {
		utils.Warn(w, r, errors.New("failed to parse originalRequestUrl"))
		return
	}

	sess.Values["oidcAuthorized"] = true
	sess.Values["oidcState"] = nil
	sess.Values["oidcOriginalRequestUrl"] = nil
	sess.Values["oidcClaims"] = claims

	err = sess.Save(r, w)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	user := model.User{
		ID:    claims[ctrl.openidConfig.Identifier].(string),
		Name:  claims["name"].(string),
		UPN:   claims["preferred_username"].(string),
		Email: claims["email"].(string),
	}
	err = ctrl.store.SaveUser(user)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	http.Redirect(w, r, originalRequestUrl, http.StatusFound)
}

func (ctrl UserCtrl) List(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindUsers(search, sort)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, http.StatusOK, "internal/views/users-many.html", map[string]any{
		"title": "Users",
		"rows":  list,
	})
}

func random(n int) string {
	// random string
	var src = rand.NewSource(time.Now().UnixNano())

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
