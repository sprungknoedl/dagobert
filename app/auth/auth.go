package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	abclientstate "github.com/aarondl/authboss-clientstate"
	"github.com/aarondl/authboss/v3"
	"github.com/aarondl/authboss/v3/auth"
	"github.com/aarondl/authboss/v3/defaults"
	_ "github.com/aarondl/authboss/v3/logout"
	_ "github.com/aarondl/authboss/v3/oauth2"
	"github.com/coreos/go-oidc"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"golang.org/x/oauth2"
)

var ab *authboss.Authboss
var SessionName = "authboss"

func Init(db *model.Store) (*authboss.Authboss, error) {
	ab = authboss.New()
	ab.Config.Storage.Server = db
	ab.Config.Storage.CookieState = abclientstate.NewCookieStorer([]byte(os.Getenv("WEB_SESSION_SECRET")), nil)
	ab.Config.Storage.SessionState = abclientstate.NewSessionStorer(SessionName, []byte(os.Getenv("WEB_SESSION_SECRET")))

	ab.Config.Paths.Mount = "/auth"

	ab.Config.Core.ViewRenderer = &Renderer{}

	ab.Config.Modules.LogoutMethod = http.MethodGet
	if os.Getenv("OIDC_ENABLED") == "true" {
		provider, err := oidc.NewProvider(context.Background(), os.Getenv("OIDC_ISSUER"))
		verifier := provider.Verifier(&oidc.Config{ClientID: os.Getenv("OIDC_CLIENT_ID")})
		if err != nil {
			return nil, err
		}

		ab.Config.Modules.OAuth2Providers = map[string]authboss.OAuth2Provider{
			"openid": {
				OAuth2Config: &oauth2.Config{
					ClientID:     os.Getenv("OIDC_CLIENT_ID"),
					ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
					RedirectURL:  os.Getenv("OIDC_CLIENT_URL"),
					Scopes:       []string{"openid", "profile", "email"},
					Endpoint:     provider.Endpoint(),
				},
				FindUserDetails: func(ctx context.Context, c oauth2.Config, t *oauth2.Token) (map[string]string, error) {
					raw, ok := t.Extra("id_token").(string)
					if !ok {
						return nil, errors.New("no id_token in oauth2 token")
					}

					idtoken, err := verifier.Verify(ctx, raw)
					if err != nil {
						return nil, err
					}

					var claims map[string]any
					err = idtoken.Claims(&claims)
					if err != nil {
						return nil, err
					}

					return fp.ApplyM(claims, func(v any) string { return fmt.Sprintf("%s", v) }), nil
				},
			},
		}
	}

	// This instantiates and uses every default implementation
	// in the Config.Core area that exist in the defaults package.
	// Just a convenient helper if you don't want to do anything fancy.
	defaults.SetCore(&ab.Config, false, false)

	err := ab.Init()
	return ab, err
}

func CurrentUser(r *http.Request) (*model.User, error) {
	abu, err := ab.CurrentUser(r)
	user, _ := abu.(*model.User)
	return user, err
}

type Renderer struct{}

// Load the given templates, will most likely be called multiple times
func (r *Renderer) Load(names ...string) error {
	slog.Debug("authboss renderer load", "names", names)
	return nil
}

// Render the given template
func (r *Renderer) Render(ctx context.Context, page string, data authboss.HTMLData) ([]byte, string, error) {
	slog.Debug("authboss renderer render", "page", page, "data", data)

	var err error
	buf := &bytes.Buffer{}

	switch page {
	case auth.PageLogin:
		err = views.Login(data).Render(ctx, buf)

	default:
		err = errors.New("template " + page + " not yet implemented")
	}

	return buf.Bytes(), "text/html", err
}
