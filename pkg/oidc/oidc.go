package oidc

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

type Config struct {
	ClientId      string   //id from the authorization service (OIDC provider)
	ClientSecret  string   //secret from the authorization service (OIDC provider)
	Issuer        url.URL  //the URL identifier for the authorization service. for example: "https://accounts.google.com" - try adding "/.well-known/openid-configuration" to the path to make sure it's correct
	ClientUrl     url.URL  //your website's/service's URL for example: "http://localhost:8081/" or "https://mydomain.com/
	Scopes        []string //OAuth scopes. If you're unsure go with: []string{oidc.ScopeOpenID, "profile", "email"}
	PostLogoutUrl url.URL  //user will be redirected to this URL after he logs out (i.e. accesses the '/logout' endpoint added in 'Init()')
	SessionName   string
}

func Middleware(e *echo.Echo, cfg Config) func(echo.HandlerFunc) echo.HandlerFunc {
	gob.Register(map[string]interface{}{})
	verifier, config := initVerifierAndConfig(cfg)

	e.GET("/logout", logoutHandler(cfg))
	e.Any("/oidc-callback", callbackHandler(cfg, verifier, config))

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			serverSession, _ := session.Get(cfg.SessionName, c)
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

func initVerifierAndConfig(cfg Config) (*oidc.IDTokenVerifier, *oauth2.Config) {
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
	return verifier, config
}

func logoutHandler(cfg Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		serverSession, _ := session.Get(cfg.SessionName, c)

		serverSession.Values["oidcAuthorized"] = false
		serverSession.Values["oidcClaims"] = nil
		serverSession.Values["oidcState"] = nil
		serverSession.Values["oidcOriginalRequestUrl"] = nil

		err := serverSession.Save(c.Request(), c.Response())
		if err != nil {
			return err
		}

		logoutUrl := cfg.Issuer
		logoutUrl.RawQuery = (url.Values{"redirect_uri": []string{cfg.PostLogoutUrl.String()}}).Encode()
		logoutUrl.Path = "protocol/openid-connect/logout"
		return c.Redirect(http.StatusFound, logoutUrl.String())
	}
}

func callbackHandler(cfg Config, verifier *oidc.IDTokenVerifier, config *oauth2.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		serverSession, _ := session.Get(cfg.SessionName, c)

		state, ok := (serverSession.Values["oidcState"]).(string)
		if !ok {
			return errors.New("get 'state' param didn't match local 'state' value")
		}

		if c.QueryParam("state") != state {
			return errors.New("get 'state' param didn't match local 'state' value")
		}

		oauth2Token, err := config.Exchange(ctx, c.QueryParam("code"))
		if err != nil {
			return err
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			return errors.New("no id_token field in oauth2 token")
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
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
