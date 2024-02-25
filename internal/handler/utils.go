package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/model"
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
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}

	return "unknown"
}

func getCase(c echo.Context) utils.CaseDTO {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return utils.CaseDTO{}
	}

	obj, err := model.GetCase(cid)
	if err != nil {
		return utils.CaseDTO{}
	}

	return utils.CaseDTO{
		ID:   obj.ID,
		Name: obj.Name,
	}
}

func ctx(c echo.Context) utils.Env {
	return utils.Env{
		Routes:      c.Echo().Reverse,
		Username:    getUser(c),
		ActiveRoute: c.Request().RequestURI,
		ActiveCase:  getCase(c),
		Search:      c.QueryParam("search"),
		Sort:        c.QueryParam("sort"),
	}
}

func formatNonZero(layout string, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(layout)
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

// random string
var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandomString(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
