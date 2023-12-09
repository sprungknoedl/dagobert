package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListEvidenceR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))
	list, err := ListEvidence(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetEvidenceR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetEvidence(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddEvidenceR(c *gin.Context) {
	cid, _ := strconv.Atoi(c.Param("cid"))

	obj := Evidence{}
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
	obj, err = SaveEvidence(c, cid, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditEvidenceR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	obj, err := GetEvidence(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Evidence{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Type = body.Type
	obj.Name = body.Name
	obj.Description = body.Description
	obj.Size = body.Size
	obj.Hash = body.Hash
	obj.Location = body.Location
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveEvidence(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteEvidenceR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cid, _ := strconv.Atoi(c.Param("cid"))
	err := DeleteEvidence(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
