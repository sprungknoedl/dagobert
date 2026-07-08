package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
)

const HeaderApiKey = "X-API-Key"

func ApiKeyMiddleware(db *model.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prefer X-API-Key (unambiguously ours, strict). If absent, treat
			// Authorization: Bearer as an api-key candidate only when it carries
			// our dgb_ prefix; any other Bearer value falls through to session
			// auth so the header isn't hijacked from other uses.
			key := r.Header.Get(HeaderApiKey)
			if key == "" {
				if bearer, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok &&
					strings.HasPrefix(bearer, model.APIKeyPrefix) {
					key = bearer
				}
			}
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// reject malformed/typo'd keys offline, before any DB query
			if !model.ValidAPIKeyFormat(key) {
				slog.Warn("api key has invalid format")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			k, err := db.GetAPIKey(model.HashAPIKey(key))
			if err != nil {
				slog.Warn("failed to get api key", "err", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// resolve the principal bound to this key's type; fail closed on
			// unknown/empty types rather than silently granting admin access
			principal, ok := model.PrincipalForAPIKeyType(k.Type)
			if !ok {
				slog.Warn("api key has unknown type", "type", k.Type)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// api key requests must not ride along browser credentials,
			// strip them before any session state is loaded
			r.Header.Del("Authorization")
			r.Header.Del("Cookie")

			// machine clients default to JSON: a keyed client that sent no
			// meaningful Accept header gets JSON responses; an explicit Accept
			// (e.g. text/html) still wins downstream
			if accept := r.Header.Get("Accept"); accept == "" || accept == "*/*" {
				r.Header.Set("Accept", "application/json")
			}

			// embed the bound principal into the request context
			ctx := context.WithValue(r.Context(), ctxKeyUser, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
