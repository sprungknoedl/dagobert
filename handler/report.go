package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sprungknoedl/dagobert/model"
	"go.arsenm.dev/pcre"
)

var pRegexp = pcre.MustCompile(`<text:p[^>]*?>{{p (.+?)}}<\/text:p>`)
var trRegexp = pcre.MustCompile(`<table:table-row[^>]*>(?:(?!<table:table-row).)*{{tr (.+?)}}.*?<\/table:table-row>`)
var expRegexp = pcre.MustCompile(`{{([^}]+)}}`)

func ListTemplate(c echo.Context) error {
	return nil
}

func ApplyTemplate(c echo.Context) error {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid case id")
	}

	obj, err := model.GetCaseFull(cid)
	if err != nil {
		return err
	}

	buf, err := GenerateReport("templates/template.odt", obj)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s - %s.odt", time.Now().Format("20060102"), obj.Name)
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Response().Header().Set("Content-Type", "application/vnd.oasis.opendocument.text")
	c.Response().Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	c.Response().WriteHeader(http.StatusOK)

	_, err = io.Copy(c.Response().Writer, buf)
	return err
}

func GenerateReport(tpl string, obj model.Case) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	zr, err := zip.OpenReader(tpl)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	zw := zip.NewWriter(buf)
	for _, item := range zr.File {
		ir, err := item.Open()
		if err != nil {
			return nil, err
		}

		header, err := zip.FileInfoHeader(item.FileInfo())
		if err != nil {
			return nil, err
		}

		header.Name = item.Name
		target, err := zw.CreateHeader(header)
		if err != nil {
			return nil, err
		}

		if item.Name == "content.xml" {
			b, err := io.ReadAll(ir)
			if err != nil {
				return nil, err
			}

			// process content.xml with text/template
			b = pRegexp.ReplaceAll(b, []byte("{{ $1 }}"))
			b = trRegexp.ReplaceAll(b, []byte("{{ $1 }}"))
			b = expRegexp.ReplaceAllFunc(b, func(x []byte) []byte {
				x = bytes.ReplaceAll(x, []byte("&quot;"), []byte("\""))
				x = bytes.ReplaceAll(x, []byte("“"), []byte("\""))
				x = bytes.ReplaceAll(x, []byte("”"), []byte("\""))
				return x
			})

			os.WriteFile("debug.xml", b, 0644)

			tpl, err := template.New("content.xml").Parse(string(b))
			if err != nil {
				return nil, err
			}

			err = tpl.Execute(target, map[string]any{
				"Case": obj,
				"Now":  time.Now(),
			})
			if err != nil {
				return nil, err
			}

		} else {
			// just copy all other files
			_, err = io.Copy(target, ir)
			if err != nil {
				return nil, err
			}
		}

		err = ir.Close()
		if err != nil {
			return nil, err
		}
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf, nil
}
