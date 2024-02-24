package handler

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
	"golang.org/x/oauth2"
)

type OpenIDConfig struct {
	ClientId      string   //id from the authorization service (OIDC provider)
	ClientSecret  string   //secret from the authorization service (OIDC provider)
	Issuer        url.URL  //the URL identifier for the authorization service. for example: "https://accounts.google.com" - try adding "/.well-known/openid-configuration" to the path to make sure it's correct
	ClientUrl     url.URL  //your website's/service's URL for example: "http://localhost:8081/" or "https://mydomain.com/
	Scopes        []string //OAuth scopes. If you're unsure go with: []string{oidc.ScopeOpenID, "profile", "email"}
	PostLogoutUrl url.URL  //user will be redirected to this URL after he logs out (i.e. accesses the '/logout' endpoint added in 'Init()')
	SessionName   string
}

type UserCtrl struct {
	openidConfig OpenIDConfig
	oauthConfig  *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

func NewUserCtrl(cfg OpenIDConfig) *UserCtrl {
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
	}
}

func (ctrl UserCtrl) AuthMiddleware(e *echo.Echo) func(echo.HandlerFunc) echo.HandlerFunc {
	gob.Register(map[string]interface{}{})
	e.GET("/logout", ctrl.Logout).Name = "logout"
	e.Any("/oidc-callback", ctrl.Callback)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			serverSession, _ := session.Get(ctrl.openidConfig.SessionName, c)
			authorized := serverSession.Values["oidcAuthorized"]
			if (authorized != nil && authorized.(bool)) || c.Request().URL.Path == "/oidc-callback" {
				return next(c)
			}

			state := RandomString(16)
			serverSession.Values["oidcAuthorized"] = false
			serverSession.Values["oidcState"] = state
			serverSession.Values["oidcOriginalRequestUrl"] = c.Request().URL.String()
			err := serverSession.Save(c.Request(), c.Response())
			if err != nil {
				log.Fatal("failed save sessions. error: " + err.Error()) // todo handle more gracefully
			}

			return c.Redirect(http.StatusFound, ctrl.oauthConfig.AuthCodeURL(state)) //redirect to authorization server
		}
	}
}

func (ctrl UserCtrl) Logout(c echo.Context) error {
	serverSession, _ := session.Get(ctrl.openidConfig.SessionName, c)

	serverSession.Values["oidcAuthorized"] = false
	serverSession.Values["oidcClaims"] = nil
	serverSession.Values["oidcState"] = nil
	serverSession.Values["oidcOriginalRequestUrl"] = nil

	err := serverSession.Save(c.Request(), c.Response())
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
	serverSession, _ := session.Get(ctrl.openidConfig.SessionName, c)

	state, ok := (serverSession.Values["oidcState"]).(string)
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

	originalRequestUrl, ok := (serverSession.Values["oidcOriginalRequestUrl"]).(string)
	if !ok {
		return errors.New("failed to parse originalRequestUrl")
	}

	serverSession.Values["oidcAuthorized"] = true
	serverSession.Values["oidcState"] = nil
	serverSession.Values["oidcOriginalRequestUrl"] = nil
	serverSession.Values["oidcClaims"] = claims

	err = serverSession.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, originalRequestUrl)
}

func (ctrl UserCtrl) ListUsers(c echo.Context) error {
	sort := c.QueryParam("sort")
	search := c.QueryParam("search")
	list, err := model.FindUsers(search, sort)
	if err != nil {
		return err
	}

	return render(c, templ.UserList(ctx(c), list))
}

func (ctrl UserCtrl) ViewUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid user id")
	}

	obj := model.User{}
	if id != 0 {
		obj, err = model.GetUser(id)
		if err != nil {
			return err
		}
	}

	return render(c, templ.UserForm(ctx(c), templ.UserDTO{
		ID:      id,
		Name:    obj.Name,
		Company: obj.Company,
		Role:    obj.Role,
		Email:   obj.Email,
		Phone:   obj.Phone,
		Notes:   obj.Notes,
	}, valid.Result{}))
}

func (ctrl UserCtrl) SaveUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil { // id == 0 is valid in this context
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid user id")
	}

	dto := templ.UserDTO{ID: id}
	if err = c.Bind(&dto); err != nil {
		return err
	}

	if vr := ValidateUser(dto); !vr.Valid() {
		return render(c, templ.UserForm(ctx(c), dto, vr))
	}

	now := time.Now()
	usr := getUser(c)
	obj := model.User{
		ID:           id,
		Name:         dto.Name,
		Company:      dto.Company,
		Role:         dto.Role,
		Email:        dto.Email,
		Phone:        dto.Phone,
		Notes:        dto.Notes,
		DateAdded:    now,
		UserAdded:    usr,
		DateModified: now,
		UserModified: usr,
	}

	if id != 0 {
		src, err := model.GetUser(id)
		if err != nil {
			return err
		}

		obj.DateAdded = src.DateAdded
		obj.UserAdded = src.UserAdded
	}

	if _, err := model.SaveUser(obj); err != nil {
		return err
	}

	return refresh(c)
}

func (ctrl UserCtrl) DeleteUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid User id")
	}

	if c.QueryParam("confirm") != "yes" {
		uri := c.Echo().Reverse("delete-user", id) + "?confirm=yes"
		return render(c, utils.Confirm(ctx(c), uri))
	}

	err = model.DeleteUser(id)
	if err != nil {
		return err
	}

	return refresh(c)
}
