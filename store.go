package main

import (
	"github.com/gin-gonic/gin"
)

// --------------------------------------
// Events
// --------------------------------------
func ListEvent(c *gin.Context) ([]Event, error) {
	var list []Event
	result := db.Order("time asc").Find(&list)
	return list, result.Error
}

func GetEvent(c *gin.Context, id int) (Event, error) {
	x := Event{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveEvent(c *gin.Context, x Event) (Event, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteEvent(c *gin.Context, id int) error {
	x := Event{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Assets
// --------------------------------------
func ListAsset(c *gin.Context) ([]Asset, error) {
	var list []Asset
	result := db.Order("name asc").Find(&list)
	return list, result.Error
}

func GetAsset(c *gin.Context, id int) (Asset, error) {
	x := Asset{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveAsset(c *gin.Context, x Asset) (Asset, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteAsset(c *gin.Context, id int) error {
	x := Asset{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Malware
// --------------------------------------
func ListMalware(c *gin.Context) ([]Malware, error) {
	var list []Malware
	result := db.Order("filename asc").Find(&list)
	return list, result.Error
}

func GetMalware(c *gin.Context, id int) (Malware, error) {
	x := Malware{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveMalware(c *gin.Context, x Malware) (Malware, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteMalware(c *gin.Context, id int) error {
	x := Asset{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Indicators
// --------------------------------------
func ListIndicator(c *gin.Context) ([]Indicator, error) {
	var list []Indicator
	result := db.Order("type asc, value asc").Find(&list)
	return list, result.Error
}

func GetIndicator(c *gin.Context, id int) (Indicator, error) {
	x := Indicator{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveIndicator(c *gin.Context, x Indicator) (Indicator, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteIndicator(c *gin.Context, id int) error {
	x := Indicator{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Users
// --------------------------------------
func ListUser(c *gin.Context) ([]User, error) {
	var list []User
	result := db.Order("short_name asc").Find(&list)
	return list, result.Error
}

func GetUser(c *gin.Context, id int) (User, error) {
	x := User{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveUser(c *gin.Context, x User) (User, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteUser(c *gin.Context, id int) error {
	x := User{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Evidences
// --------------------------------------
func ListEvidence(c *gin.Context) ([]Evidence, error) {
	var list []Evidence
	result := db.Order("name asc").Find(&list)
	return list, result.Error
}

func GetEvidence(c *gin.Context, id int) (Evidence, error) {
	x := Evidence{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveEvidence(c *gin.Context, x Evidence) (Evidence, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteEvidence(c *gin.Context, id int) error {
	x := Evidence{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Tasks
// --------------------------------------
func ListTask(c *gin.Context) ([]Task, error) {
	var list []Task
	result := db.Order("date_due asc, type asc, task asc").Find(&list)
	return list, result.Error
}

func GetTask(c *gin.Context, id int) (Task, error) {
	x := Task{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveTask(c *gin.Context, x Task) (Task, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteTask(c *gin.Context, id int) error {
	x := Task{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Notes
// --------------------------------------
func ListNote(c *gin.Context) ([]Note, error) {
	var list []Note
	result := db.Order("category asc, title asc").Find(&list)
	return list, result.Error
}

func GetNote(c *gin.Context, id int) (Note, error) {
	x := Note{}
	result := db.First(&x, id)
	return x, result.Error
}

func SaveNote(c *gin.Context, x Note) (Note, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteNote(c *gin.Context, id int) error {
	x := Note{}
	return db.Delete(&x, id).Error
}
