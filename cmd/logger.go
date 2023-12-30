package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

var PrettyLogger = middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
	LogStatus:     true,
	LogURI:        true,
	LogError:      true,
	LogRemoteIP:   true,
	LogMethod:     true,
	LogLatency:    true,
	HandleError:   true, // forwards error to the global error handler, so it can decide appropriate status code
	LogValuesFunc: logger,
})

func statusColor(v middleware.RequestLoggerValues) string {
	switch {
	case v.Status >= http.StatusContinue && v.Status < http.StatusOK:
		return white
	case v.Status >= http.StatusOK && v.Status < http.StatusMultipleChoices:
		return green
	case v.Status >= http.StatusMultipleChoices && v.Status < http.StatusBadRequest:
		return white
	case v.Status >= http.StatusBadRequest && v.Status < http.StatusInternalServerError:
		return yellow
	default:
		return red
	}
}

func methodColor(v middleware.RequestLoggerValues) string {
	switch v.Method {
	case http.MethodGet:
		return blue
	case http.MethodPost:
		return cyan
	case http.MethodPut:
		return yellow
	case http.MethodDelete:
		return red
	case http.MethodPatch:
		return green
	case http.MethodHead:
		return magenta
	case http.MethodOptions:
		return white
	default:
		return reset
	}
}

func logger(c echo.Context, v middleware.RequestLoggerValues) error {
	statusColor := statusColor(v)
	methodColor := methodColor(v)

	if v.Latency > time.Minute {
		v.Latency = v.Latency.Truncate(time.Second)
	}

	errmsg := ""
	if v.Status >= http.StatusInternalServerError && v.Error != nil {
		errmsg = fmt.Sprintf("\n                    |%s ERR %s| %v", statusColor, reset, v.Error)
	}
	_, err := fmt.Printf("%s |%s %3d %s| %13v | %15s |%s %-7s %s %q %s\n",
		v.StartTime.Format("2006/01/02 15:04:05"),
		statusColor, v.Status, reset,
		v.Latency,
		v.RemoteIP,
		methodColor, v.Method, reset,
		v.URI,
		errmsg)

	return err
}
