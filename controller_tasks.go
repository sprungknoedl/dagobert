package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListTaskR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
	list, err := ListTask(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetTaskR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetTask(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddTaskR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))

	obj := Task{}
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
	obj, err = SaveTask(c, cid, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditTaskR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetTask(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Task{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Type = body.Type
	obj.Task = body.Task
	obj.Done = body.Done
	obj.Owner = body.Owner
	obj.DateAdded = body.DateAdded
	obj.DateDue = body.DateDue
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveTask(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteTaskR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	err := DeleteTask(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
