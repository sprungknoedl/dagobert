package handler

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/model"
	"golang.org/x/oauth2"
)

type OpenIDConfig struct {
	ClientId      string   //id from the authorization service (OIDC provider)
	ClientSecret  string   //secret from the authorization service (OIDC provider)
	ClientUrl     url.URL  //your website's/service's URL for example: "http://localhost:8081/" or "https://mydomain.com/
	Issuer        url.URL  //the URL identifier for the authorization service. for example: "https://accounts.google.com" - try adding "/.well-known/openid-configuration" to the path to make sure it's correct
	Scopes        []string //OAuth scopes. If you're unsure go with: []string{oidc.ScopeOpenID, "profile", "email"}
	Identifier    string   // name of the openid claim used to securely identify a user (e.g. "sub").
	PostLogoutUrl url.URL  //user will be redirected to this URL after he logs out (i.e. accesses the '/logout' endpoint added in 'Init()')
	SessionName   string
}

type UserCtrl struct {
	openidConfig OpenIDConfig
	oauthConfig  *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	store        model.UserStore
}

func NewUserCtrl(store model.UserStore, cfg OpenIDConfig) *UserCtrl {
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

func (ctrl UserCtrl) Protect(e *echo.Echo) func(echo.HandlerFunc) echo.HandlerFunc {
	gob.Register(map[string]interface{}{})
	e.GET("/logout", ctrl.Logout).Name = "logout"
	e.Any("/oidc-callback", ctrl.Callback)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, _ := session.Get(ctrl.openidConfig.SessionName, c)
			authorized := sess.Values["oidcAuthorized"]
			if (authorized != nil && authorized.(bool)) || c.Request().URL.Path == "/oidc-callback" {
				return next(c)
			}

			state := random(16)
			sess.Values["oidcAuthorized"] = false
			sess.Values["oidcState"] = state
			sess.Values["oidcOriginalRequestUrl"] = c.Request().URL.String()
			err := sess.Save(c.Request(), c.Response())
			if err != nil {
				log.Fatal("failed save sessions. error: " + err.Error()) // todo handle more gracefully
			}

			return c.Redirect(http.StatusFound, ctrl.oauthConfig.AuthCodeURL(state)) //redirect to authorization server
		}
	}
}

func (ctrl UserCtrl) Logout(c echo.Context) error {
	sess, _ := session.Get(ctrl.openidConfig.SessionName, c)
	sess.Values["oidcAuthorized"] = false
	sess.Values["oidcClaims"] = nil
	sess.Values["oidcState"] = nil
	sess.Values["oidcOriginalRequestUrl"] = nil

	err := sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	logoutUrl := ctrl.openidConfig.Issuer
	logoutUrl.RawQuery = (url.Values{"redirect_uri": []string{ctrl.openidConfig.PostLogoutUrl.String()}}).Encode()
	logoutUrl.Path = "protocol/openid-connect/logout"
	return c.Redirect(http.StatusFound, logoutUrl.String())
}

func (ctrl UserCtrl) Callback(c echo.Context) error {
	ctx := c.Request().Context()
	sess, _ := session.Get(ctrl.openidConfig.SessionName, c)

	state, ok := (sess.Values["oidcState"]).(string)
	if !ok {
		return errors.New("get 'state' param didn't match local 'state' value")
	}

	if c.QueryParam("state") != state {
		return errors.New("get 'state' param didn't match local 'state' value")
	}

	oauth2Token, err := ctrl.oauthConfig.Exchange(ctx, c.QueryParam("code"))
	if err != nil {
		return err
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return errors.New("no id_token field in oauth2 token")
	}

	idToken, err := ctrl.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return err
	}

	var claims map[string]interface{}
	err = idToken.Claims(&claims)
	if err != nil {
		return err
	}

	originalRequestUrl, ok := (sess.Values["oidcOriginalRequestUrl"]).(string)
	if !ok {
		return errors.New("failed to parse originalRequestUrl")
	}

	sess.Values["oidcAuthorized"] = true
	sess.Values["oidcState"] = nil
	sess.Values["oidcOriginalRequestUrl"] = nil
	sess.Values["oidcClaims"] = claims

	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	user := model.User{
		ID:    claims[ctrl.openidConfig.Identifier].(string),
		Name:  claims["name"].(string),
		UPN:   claims["preferred_username"].(string),
		Email: claims["email"].(string),
	}
	_, err = ctrl.store.SaveUser(user)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, originalRequestUrl)
}

func (ctrl UserCtrl) List(c echo.Context) error {
	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := ctrl.store.FindUsers(search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.UserList(ctx(c), list))
}
