package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListUserR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
	list, err := ListUser(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetUserR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetUser(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddUserR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))

	obj := User{}
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
	obj, err = SaveUser(c, cid, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditUserR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetUser(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := User{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.ShortName = body.ShortName
	obj.FullName = body.FullName
	obj.Company = body.Company
	obj.Role = body.Role
	obj.Email = body.Email
	obj.Phone = body.Phone
	obj.Notes = body.Notes
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveUser(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteUserR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	err := DeleteUser(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
