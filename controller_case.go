package main

import (
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

func GetCaseR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
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
	cid, _ := strconv.Atoi(c.Param("cid"))
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
	cid, _ := strconv.Atoi(c.Param("cid"))
	err := DeleteCase(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
