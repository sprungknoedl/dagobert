package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// --------------------------------------
// Events
// --------------------------------------
func ListEventR(c *gin.Context) {
	list, err := ListEvent(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetEventR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetEvent(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddEventR(c *gin.Context) {
	obj := Event{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	if _, err := SaveEvent(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditEventR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetEvent(c, id)
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

	if _, err := SaveEvent(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteEventR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteEvent(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Assets
// --------------------------------------
func ListAssetR(c *gin.Context) {
	list, err := ListAsset(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetAssetR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetAsset(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddAssetR(c *gin.Context) {
	obj := Asset{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	if _, err := SaveAsset(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditAssetR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetAsset(c, id)
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

	if _, err := SaveAsset(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteAssetR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteAsset(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Malware
// --------------------------------------
func ListMalwareR(c *gin.Context) {
	list, err := ListMalware(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetMalwareR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetMalware(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddMalwareR(c *gin.Context) {
	obj := Malware{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	if _, err := SaveMalware(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditMalwareR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetMalware(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Malware{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Filename = body.Filename
	obj.Filepath = body.Filepath
	obj.CDate = body.CDate
	obj.MDate = body.MDate
	obj.System = body.System
	obj.Hash = body.Hash
	obj.Notes = body.Notes

	if _, err := SaveMalware(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteMalwareR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteMalware(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Indicators
// --------------------------------------
func ListIndicatorR(c *gin.Context) {
	list, err := ListIndicator(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetIndicatorR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetIndicator(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddIndicatorR(c *gin.Context) {
	obj := Indicator{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	if _, err := SaveIndicator(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditIndicatorR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetIndicator(c, id)
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

	if _, err := SaveIndicator(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func DeleteIndicatorR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteIndicator(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Users
// --------------------------------------
func ListUserR(c *gin.Context) {
	list, err := ListUser(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetUserR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetUser(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddUserR(c *gin.Context) {
	obj := User{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	obj, err = SaveUser(c, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditUserR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetUser(c, id)
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

	if _, err := SaveUser(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteUserR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteUser(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Evidences
// --------------------------------------
func ListEvidenceR(c *gin.Context) {
	list, err := ListEvidence(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetEvidenceR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetEvidence(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddEvidenceR(c *gin.Context) {
	obj := Evidence{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	obj, err = SaveEvidence(c, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditEvidenceR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetEvidence(c, id)
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

	if _, err := SaveEvidence(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteEvidenceR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteEvidence(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Tasks
// --------------------------------------
func ListTaskR(c *gin.Context) {
	list, err := ListTask(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetTaskR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetTask(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddTaskR(c *gin.Context) {
	obj := Task{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	obj, err = SaveTask(c, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditTaskR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetTask(c, id)
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

	if _, err := SaveTask(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteTaskR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteTask(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}

// --------------------------------------
// Notes
// --------------------------------------
func ListNoteR(c *gin.Context) {
	list, err := ListNote(c)
	if err != nil {
		c.String(http.StatusBadRequest, "list: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, list)
}

func GetNoteR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetNote(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, obj)
}

func AddNoteR(c *gin.Context) {
	obj := Note{}
	err := c.BindJSON(&obj)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	obj, err = SaveNote(c, obj)
	if err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}

	c.JSON(http.StatusCreated, obj)
}

func EditNoteR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	obj, err := GetNote(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "get: %s", err.Error())
		return
	}

	body := Note{}
	err = c.BindJSON(&body)
	if err != nil {
		c.String(http.StatusBadRequest, "bind: %s", err.Error())
		return
	}

	// Only copy over fields we wan't to be editable
	obj.Title = body.Title
	obj.Category = body.Category
	obj.Description = body.Description

	if _, err := SaveNote(c, obj); err != nil {
		c.String(http.StatusBadRequest, "save: %s", err.Error())
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteNoteR(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := DeleteNote(c, id)
	if err != nil {
		c.String(http.StatusBadRequest, "delete: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
