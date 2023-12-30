package handler

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/components/utils"
)

const SessionName = "session"

func Empty(c echo.Context) error {
	return render(c, utils.DialogPlaceholder())
}

func render(c echo.Context, component templ.Component) error {
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func refresh(c echo.Context) error {
	c.Response().Header().Add("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}

func getUser(c echo.Context) string {
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
