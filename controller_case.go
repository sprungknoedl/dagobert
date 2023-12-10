package main

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListCaseR(c *gin.Context) {
	list, err := ListCase(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func ExportCaseCsvR(c *gin.Context) {
	list, err := ListCase(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Disposition", "attachment; filename=\"cases.csv\"")

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"ID", "Name", "Classification", "Summary"})
	for _, e := range list {
		w.Write([]string{strconv.FormatInt(e.ID, 10), e.Name, e.Classification, e.Summary})
	}
	w.Flush()
}

func GetCaseR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	obj, err := GetCase(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddCaseR(c *gin.Context) {
	obj := Case{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	username := GetUsername(c)
	obj.DateAdded = time.Now()
	obj.UserAdded = username
	obj.DateModified = time.Now()
	obj.UserModified = username
	if _, err := SaveCase(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditCaseR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	obj, err := GetCase(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Case{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Name = body.Name
	obj.Classification = body.Classification
	obj.Summary = body.Summary
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveCase(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteCaseR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	err := DeleteCase(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
