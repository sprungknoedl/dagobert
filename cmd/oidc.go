package main

import (
	"context"
	"encoding/gob"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

const SessionName = "default"

type InitParams struct {
	ClientId      string   //id from the authorization service (OIDC provider)
	ClientSecret  string   //secret from the authorization service (OIDC provider)
	Issuer        url.URL  //the URL identifier for the authorization service. for example: "https://accounts.google.com" - try adding "/.well-known/openid-configuration" to the path to make sure it's correct
	ClientUrl     url.URL  //your website's/service's URL for example: "http://localhost:8081/" or "https://mydomain.com/
	Scopes        []string //OAuth scopes. If you're unsure go with: []string{oidc.ScopeOpenID, "profile", "email"}
	PostLogoutUrl url.URL  //user will be redirected to this URL after he logs out (i.e. accesses the '/logout' endpoint added in 'Init()')
}

func OIDC(e *echo.Echo, i InitParams) func(echo.HandlerFunc) echo.HandlerFunc {
	gob.Register(map[string]interface{}{})
	verifier, config := initVerifierAndConfig(i)

	e.GET("/logout", logoutHandler(i))
	e.Any("/oidc-callback", callbackHandler(i, verifier, config))

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			serverSession, _ := session.Get(SessionName, c)
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

			return c.Redirect(http.StatusFound, config.AuthCodeURL(state)) //redirect to authorization server
		}
	}
}

func initVerifierAndConfig(i InitParams) (*oidc.IDTokenVerifier, *oauth2.Config) {
	providerCtx := context.Background()
	provider, err := oidc.NewProvider(providerCtx, i.Issuer.String())
	if err != nil {
		log.Fatalf("Failed to init OIDC provider. Error: %v \n", err.Error())
	}
	oidcConfig := &oidc.Config{
		ClientID: i.ClientId,
	}
	verifier := provider.Verifier(oidcConfig)
	endpoint := provider.Endpoint()
	i.ClientUrl.Path = "oidc-callback"
	config := &oauth2.Config{
		ClientID:     i.ClientId,
		ClientSecret: i.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  i.ClientUrl.String(),
		Scopes:       i.Scopes,
	}
	return verifier, config
}

func logoutHandler(i InitParams) echo.HandlerFunc {
	return func(c echo.Context) error {
		serverSession, _ := session.Get(SessionName, c)

		serverSession.Values["oidcAuthorized"] = false
		serverSession.Values["oidcClaims"] = nil
		serverSession.Values["oidcState"] = nil
		serverSession.Values["oidcOriginalRequestUrl"] = nil

		err := serverSession.Save(c.Request(), c.Response())
		if err != nil {
			return err
		}

		logoutUrl := i.Issuer
		logoutUrl.RawQuery = (url.Values{"redirect_uri": []string{i.PostLogoutUrl.String()}}).Encode()
		logoutUrl.Path = "protocol/openid-connect/logout"
		return c.Redirect(http.StatusFound, logoutUrl.String())
	}
}

func callbackHandler(i InitParams, verifier *oidc.IDTokenVerifier, config *oauth2.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		serverSession, _ := session.Get(SessionName, c)

		state, ok := (serverSession.Values["oidcState"]).(string)
		if handleOk(c, i, ok, "failed to parse state") {
			return errors.New("get 'state' param didn't match local 'state' value")
		}

		if handleOk(c, i, c.QueryParam("state") == state, "get 'state' param didn't match local 'state' value") {
			return errors.New("get 'state' param didn't match local 'state' value")
		}

		oauth2Token, err := config.Exchange(ctx, c.QueryParam("code"))
		if handleError(c, i, err, "failed to exchange token") {
			return err
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if handleOk(c, i, ok, "no id_token field in oauth2 token") {
			return errors.New("no id_token field in oauth2 token")
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if handleError(c, i, err, "failed to verify id token") {
			return err
		}

		var claims map[string]interface{}
		err = idToken.Claims(&claims)
		if handleError(c, i, err, "failed to parse id token") {
			return err
		}

		originalRequestUrl, ok := (serverSession.Values["oidcOriginalRequestUrl"]).(string)
		if handleOk(c, i, ok, "failed to parse originalRequestUrl") {
			return errors.New("failed to parse originalRequestUrl")
		}

		serverSession.Values["oidcAuthorized"] = true
		serverSession.Values["oidcState"] = nil
		serverSession.Values["oidcOriginalRequestUrl"] = nil
		serverSession.Values["oidcClaims"] = claims

		err = serverSession.Save(c.Request(), c.Response())
		if handleError(c, i, err, "failed save sessions.") {
			return err
		}

		return c.Redirect(http.StatusFound, originalRequestUrl)
	}
}

func handleError(c echo.Context, i InitParams, err error, message string) bool {
	return err != nil
}

func handleOk(c echo.Context, i InitParams, ok bool, message string) bool {
	if ok {
		return false
	}
	return handleError(c, i, errors.New("not ok"), message)
}

// random string
var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandomString(n int) string {
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
