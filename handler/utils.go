package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/utils"
)

const SessionName = "default"

func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	c.Response().Header().Add("HX-Retarget", "#errors")
	c.Response().Header().Add("HX-Reswap", "beforeend")
	c.Response().WriteHeader(http.StatusOK)

	if he, ok := err.(*echo.HTTPError); ok {
		render(c, utils.WarningNotification(he))
	} else {
		render(c, utils.ErrorNotification(err))
	}
}

func render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func refresh(c echo.Context) error {
	c.Response().Header().Add("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}

func getUser(c echo.Context) string {
	sess, _ := session.Get(SessionName, c)
	claims, _ := sess.Values["oidcClaims"].(map[string]interface{})
	if email, ok := claims["email"].(string); ok {
		return email
	}

	return "unknown"
}

func getCase(c echo.Context) utils.CaseDTO {
	sess, _ := session.Get(SessionName, c)
	kase, _ := sess.Values["activeCase"].(utils.CaseDTO)
	return kase
}

func ctx(c echo.Context) utils.Env {
	return utils.Env{
		Routes:      c.Echo().Reverse,
		Username:    getUser(c),
		ActiveRoute: c.Request().RequestURI,
		ActiveCase:  getCase(c),
	}
}

func importHelper(c echo.Context, uri string, numFields int, cb func(c echo.Context, rec []string) error) error {
	if c.Request().Method == http.MethodGet {
		return render(c, utils.Import(ctx(c), uri))
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	fr, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	r := csv.NewReader(fr)
	r.FieldsPerRecord = numFields
	r.Read() // skip header

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		err = cb(c, rec)
		if err != nil {
			return err
		}
	}

	return refresh(c)
}
