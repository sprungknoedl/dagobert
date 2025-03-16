package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("599: Recovered From Panic",
					"err", err,
					"raddr", r.RemoteAddr,
					"method", r.Method,
					"url", r.URL)

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
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
