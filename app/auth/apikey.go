package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/sprungknoedl/dagobert/app/model"
)

const HeaderApiKey = "X-API-Key"

func ApiKeyMiddleware(db *model.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(HeaderApiKey)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			_, err := db.GetKey(key)
			if err != nil {
				slog.Warn("failed to get api key", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// api key requests must not ride along browser credentials,
			// strip them before any session state is loaded
			r.Header.Del("Authorization")
			r.Header.Del("Cookie")

			// embed system user into session
			r = r.WithContext(context.WithValue(r.Context(), authboss.CTXKeyUser, &model.SystemUser))
			next.ServeHTTP(w, r)
		})
	}
}
