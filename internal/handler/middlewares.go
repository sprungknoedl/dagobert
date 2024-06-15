package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/pkg/tty"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("| %s | %v", tty.Red("PAN"), err)
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

		statusColor := statusColor(lw.Status)
		methodColor := methodColor(r.Method)

		if duration > time.Minute {
			duration = duration.Truncate(time.Second)
		}

		log.Printf("| %-3s | %13v | %15s | %s %q",
			statusColor(fmt.Sprintf("%3d", lw.Status)),
			duration,
			r.RemoteAddr,
			methodColor(fmt.Sprintf("%-7s", r.Method)),
			r.URL)
	})
}

// LoggingResponseWriter struct is used to log the response
type LoggingResponseWriter struct {
	w      http.ResponseWriter
	Bytes  int
	Status int
}

func (w *LoggingResponseWriter) Write(buf []byte) (int, error) {
	n, err := w.w.Write(buf)
	w.Bytes += n
	return n, err
}

func (w *LoggingResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *LoggingResponseWriter) WriteHeader(statusCode int) {
	w.Status = statusCode
	w.w.WriteHeader(statusCode)
}

func statusColor(status int) tty.Fn {
	switch {
	case status >= http.StatusContinue && status < http.StatusOK:
		return tty.White
	case status >= http.StatusOK && status < http.StatusMultipleChoices:
		return tty.Green
	case status >= http.StatusMultipleChoices && status < http.StatusBadRequest:
		return tty.White
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		return tty.Yellow
	default:
		return tty.Red
	}
}

func methodColor(method string) tty.Fn {
	switch method {
	case http.MethodGet:
		return tty.Blue
	case http.MethodPost:
		return tty.Cyan
	case http.MethodPut:
		return tty.Yellow
	case http.MethodDelete:
		return tty.Red
	case http.MethodPatch:
		return tty.Green
	case http.MethodHead:
		return tty.Magenta
	case http.MethodOptions:
		return tty.White
	default:
		return tty.White
	}
}
