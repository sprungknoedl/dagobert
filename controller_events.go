package main

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListEventR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	list, err := ListEvent(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func ExportEventCsvR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	list, err := ListEvent(c, cid)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Disposition", "attachment; filename=\"events.csv\"")

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"Time", "Type", "Event System", "Direction", "Remote System", "Event", "Raw"})
	for _, e := range list {
		w.Write([]string{e.Time.Format(time.RFC3339), e.Type, e.AssetA, e.Direction, e.AssetB, e.Event, e.Raw})
	}
	w.Flush()
}

func GetEventR(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	obj, err := GetEvent(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddEventR(c *gin.Context) {
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)

	obj := Event{}
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
	if _, err := SaveEvent(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditEventR(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	obj, err := GetEvent(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Event{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Time = body.Time
	obj.Type = body.Type
	obj.AssetA = body.AssetA
	obj.AssetB = body.AssetB
	obj.Direction = body.Direction
	obj.Event = body.Event
	obj.Raw = body.Raw
	obj.DateModified = time.Now()
	obj.UserModified = GetUsername(c)

	if _, err := SaveEvent(c, cid, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteEventR(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	cid, _ := strconv.ParseInt(c.Param("cid"), 10, 64)
	err := DeleteEvent(c, cid, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
