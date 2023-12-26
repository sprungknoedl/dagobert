package main

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// --------------------------------------
// Cases
// --------------------------------------
func ListCase(c *gin.Context) ([]Case, error) {
	var list []Case
	result := db.Order("name asc").Find(&list)
	return list, result.Error
}

func GetCase(c *gin.Context, id int64) (Case, error) {
	x := Case{}
	result := db.First(&x, id)
	return x, result.Error
}

func GetCaseFull(c *gin.Context, id int64) (Case, error) {
	x := Case{}
	result := db.
		Preload(clause.Associations).
		First(&x, id)
	return x, result.Error
}

func SaveCase(c *gin.Context, x Case) (Case, error) {
	result := db.Save(&x)
	return x, result.Error
}

func DeleteCase(c *gin.Context, id int64) error {
	x := Case{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Events
// --------------------------------------
func ListEvent(c *gin.Context, cid int64) ([]Event, error) {
	var list []Event
	result := db.Order("time asc").
		Where("case_id = ?", cid).
		Find(&list)
	return list, result.Error
}

func GetEvent(c *gin.Context, cid int64, id int64) (Event, error) {
	x := Event{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveEvent(c *gin.Context, cid int64, x Event) (Event, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteEvent(c *gin.Context, cid int64, id int64) error {
	x := Event{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Assets
// --------------------------------------
func ListAsset(c *gin.Context, cid int64) ([]Asset, error) {
	var list []Asset
	result := db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func GetAsset(c *gin.Context, cid int64, id int64) (Asset, error) {
	x := Asset{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveAsset(c *gin.Context, cid int64, x Asset) (Asset, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteAsset(c *gin.Context, cid int64, id int64) error {
	x := Asset{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Malware
// --------------------------------------
func ListMalware(c *gin.Context, cid int64) ([]Malware, error) {
	var list []Malware
	result := db.Where("case_id = ?", cid).
		Order("filename asc").
		Find(&list)
	return list, result.Error
}

func GetMalware(c *gin.Context, cid int64, id int64) (Malware, error) {
	x := Malware{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveMalware(c *gin.Context, cid int64, x Malware) (Malware, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteMalware(c *gin.Context, cid int64, id int64) error {
	x := Asset{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Indicators
// --------------------------------------
func ListIndicator(c *gin.Context, cid int64) ([]Indicator, error) {
	var list []Indicator
	result := db.
		Where("case_id = ?", cid).
		Order("type asc, value asc").
		Find(&list)
	return list, result.Error
}

func GetIndicator(c *gin.Context, cid int64, id int64) (Indicator, error) {
	x := Indicator{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveIndicator(c *gin.Context, cid int64, x Indicator) (Indicator, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteIndicator(c *gin.Context, cid int64, id int64) error {
	x := Indicator{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Users
// --------------------------------------
func ListUser(c *gin.Context, cid int64) ([]User, error) {
	var list []User
	result := db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func GetUser(c *gin.Context, cid int64, id int64) (User, error) {
	x := User{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveUser(c *gin.Context, cid int64, x User) (User, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteUser(c *gin.Context, cid int64, id int64) error {
	x := User{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Evidences
// --------------------------------------
func ListEvidence(c *gin.Context, cid int64) ([]Evidence, error) {
	var list []Evidence
	result := db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func GetEvidence(c *gin.Context, cid int64, id int64) (Evidence, error) {
	x := Evidence{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveEvidence(c *gin.Context, cid int64, x Evidence) (Evidence, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteEvidence(c *gin.Context, cid int64, id int64) error {
	x := Evidence{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Tasks
// --------------------------------------
func ListTask(c *gin.Context, cid int64) ([]Task, error) {
	var list []Task
	result := db.
		Where("case_id = ?", cid).
		Order("date_due asc, type asc, task asc").
		Find(&list)
	return list, result.Error
}

func GetTask(c *gin.Context, cid int64, id int64) (Task, error) {
	x := Task{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveTask(c *gin.Context, cid int64, x Task) (Task, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteTask(c *gin.Context, cid int64, id int64) error {
	x := Task{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Notes
// --------------------------------------
func ListNote(c *gin.Context, cid int64) ([]Note, error) {
	var list []Note
	result := db.
		Where("case_id = ?", cid).
		Order("category asc, title asc").
		Find(&list)
	return list, result.Error
}

func GetNote(c *gin.Context, cid int64, id int64) (Note, error) {
	x := Note{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveNote(c *gin.Context, cid int64, x Note) (Note, error) {
	result := db.
		Where("case_id = ?", cid).
		Save(&x)
	return x, result.Error
}

func DeleteNote(c *gin.Context, cid int64, id int64) error {
	x := Note{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
