package auth

import (
	"context"
	"log/slog"
	"net/http"

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

			k, err := db.GetKey(key)
			if err != nil {
				slog.Warn("failed to get api key", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// resolve the principal bound to this key's type; fail closed on
			// unknown/empty types rather than silently granting admin access
			principal, ok := model.PrincipalForKeyType(k.Type)
			if !ok {
				slog.Warn("api key has unknown type", "type", k.Type)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// api key requests must not ride along browser credentials,
			// strip them before any session state is loaded
			r.Header.Del("Authorization")
			r.Header.Del("Cookie")

			// embed the bound principal into the request context and mark this
			// as a non-interactive API request so handlers can tailor responses
			ctx := context.WithValue(r.Context(), ctxKeyUser, principal)
			ctx = context.WithValue(ctx, ctxKeyAPI, true)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
