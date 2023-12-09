package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListIndicatorR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
	list, err := ListIndicator(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetIndicatorR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetIndicator(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddIndicatorR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))

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
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
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
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	err := DeleteIndicator(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
