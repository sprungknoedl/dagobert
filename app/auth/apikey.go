package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

const HeaderApiKey = "X-API-Key"

func ApiKeyMiddleware(ab *authboss.Authboss, db *model.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(HeaderApiKey)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			_, err := db.GetKey(key)
			if err != nil {
				traceid := fp.Random(32)
				slog.Warn("failed to get api key", "err", err, "trace", traceid)
				w.WriteHeader(http.StatusUnauthorized)
				// TODO add error and traceid
				return
			}

			// strip cookie and authorization header
			r.Header.Del("Authorization")
			r.Header.Del("Cookie")

			// embed system user into session
			r = r.WithContext(context.WithValue(r.Context(), authboss.CTXKeyUser, &model.SystemUser))
			next.ServeHTTP(w, r)
		})
	}
}
