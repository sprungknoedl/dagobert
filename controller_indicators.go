package main

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListIndicatorR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	list, err := ListIndicator(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func ExportIndicatorCsvR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	list, err := ListIndicator(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Disposition", "attachment; filename=\"indicators.csv\"")

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"Type", "Value", "TLP", "Description", "Source"})
	for _, e := range list {
		w.Write([]string{e.Type, e.Value, e.TLP, e.Description, e.Source})
	}
	w.Flush()
}

func GetIndicatorR(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	obj, err := GetIndicator(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddIndicatorR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)

	obj := Indicator{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	username := GetUsername(c)
	obj.CaseID = cid
	obj.DateAdded = time.Now()
	obj.UserAdded = username
	obj.DateModified = time.Now()
	obj.UserModified = username
	if _, err := SaveIndicator(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditIndicatorR(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	obj, err := GetIndicator(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Indicator{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Type = body.Type
	obj.Value = body.Value
	obj.TLP = body.TLP
	obj.Description = body.Description
	obj.Source = body.Source
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveIndicator(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteIndicatorR(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	err := DeleteIndicator(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}