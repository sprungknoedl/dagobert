package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListAssetR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
	list, err := ListAsset(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetAssetR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetAsset(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddAssetR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))

	obj := Asset{}
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
	if _, err := SaveAsset(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditAssetR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetAsset(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Asset{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Type = body.Type
	obj.Name = body.Name
	obj.IP = body.IP
	obj.Description = body.Description
	obj.Compromised = body.Compromised
	obj.Analysed = body.Analysed
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveAsset(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteAssetR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	err := DeleteAsset(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
