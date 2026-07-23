package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/views"
)

// contentSecurityPolicy is a strict policy for an app that renders attacker-
// influenced strings (case notes, indicator values, evidence metadata) back
// into the page: no 'unsafe-inline'/'unsafe-eval' for scripts. Inline styles
// are allowed since they're presentational-only and not a script-injection
// vector.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self'; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("599: Recovered From Panic",
					"err", err,
					"raddr", r.RemoteAddr,
					"method", r.Method,
					"url", r.URL)
				slog.Error("stack trace:\n" + string(debug.Stack()))

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets baseline hardening headers on every response: deny
// framing (clickjacking), stop content-type sniffing (matters for the served
// evidence and malware files), keep the Referer from leaking to other
// origins, and apply a strict CSP. HSTS is only sent when WEB_SECURE has not
// disabled TLS-only mode (mirrors Session.Cookie.Secure), since it's only
// meaningful over TLS while plain-HTTP dev is supported. Headers are set
// before the handler runs so they apply to error responses too.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		if os.Getenv("WEB_SECURE") != "false" {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// ThemeMiddleware resolves the theme cookie once per request and stores the
// server-rendered data-theme value in context: "dagobert" (light),
// "dagobert-dark" (dark), or "" when the cookie is absent (Auto — layout()
// omits data-theme so DaisyUI's prefersdark theme follows the OS preference).
func ThemeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		theme := ""
		if c, err := r.Cookie("theme"); err == nil {
			switch c.Value {
			case "light":
				theme = "dagobert"
			case "dark":
				theme = "dagobert-dark"
			}
		}

		ctx := context.WithValue(r.Context(), views.ThemeCtxKey, theme)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &LoggingResponseWriter{w: w, Status: http.StatusOK}

		start := time.Now()
		next.ServeHTTP(lw, r)
		duration := time.Since(start)

		slog.Info(strconv.Itoa(lw.Status)+": "+http.StatusText(lw.Status),
			slog.Duration("duration", duration),
			"raddr", r.RemoteAddr,
			"method", r.Method,
			"url", r.URL)
	})
}

// LoggingResponseWriter struct is used to log the response
type LoggingResponseWriter struct {
	w             http.ResponseWriter
	Bytes         int
	Status        int
	HeaderWritten bool
}

func (w *LoggingResponseWriter) Write(buf []byte) (int, error) {
	w.HeaderWritten = true
	n, err := w.w.Write(buf)
	w.Bytes += n
	return n, err
}

func (w *LoggingResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *LoggingResponseWriter) WriteHeader(statusCode int) {
	w.w.WriteHeader(statusCode)

	if !w.HeaderWritten {
		w.Status = statusCode
		w.HeaderWritten = true
	}
}

func (w *LoggingResponseWriter) Unwrap() http.ResponseWriter {
	return w.w
}
