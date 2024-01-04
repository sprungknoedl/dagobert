package model

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var db *gorm.DB

func InitDatabase(dburl string) {
	var err error
	db, err = gorm.Open(sqlite.Open(dburl), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&Case{},
		&Event{},
		&Asset{},
		&Malware{},
		&Indicator{},
		&User{},
		&Evidence{},
		&Task{},
		&Note{},
	)
	if err != nil {
		log.Fatalf("failed to migrate db: %v", err)
	}
}

// --------------------------------------
// Cases
// --------------------------------------
func FindCases(term string) ([]Case, error) {
	var list []Case
	result := db.Order("name asc").
		Where("instr(name, ?) > 0", term).
		Or("instr(classification, ?) > 0", term).
		Or("instr(severity, ?) > 0", term).
		Or("instr(outcome, ?) > 0", term).
		Or("instr(summary, ?) > 0", term).
		Find(&list)
	return list, result.Error
}

func ListCases() ([]Case, error) {
	var list []Case
	result := db.Order("name asc").Find(&list)
	return list, result.Error
}

func GetCase(id int64) (Case, error) {
	x := Case{}
	result := db.First(&x, id)
	return x, result.Error
}

func GetCaseFull(id int64) (Case, error) {
	x := Case{}
	result := db.
		Preload(clause.Associations).
		First(&x, id)
	return x, result.Error
}

func SaveCase(x Case) (Case, error) {
	x.CRC = HashFields(
		x.Name,
		x.Closed,
		x.Classification,
		x.Severity,
		x.Outcome,
		x.Summary,
	)

	result := db.
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteCase(id int64) error {
	x := Case{}
	return db.Delete(&x, id).Error
}

// --------------------------------------
// Events
// --------------------------------------
func FindEvents(cid int64, term string) ([]Event, error) {
	var list []Event
	result := db.Order("time asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", term).
			Or("instr(asset_a, ?) > 0", term).
			Or("instr(asset_b, ?) > 0", term).
			Or("instr(event, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func ListEvents(cid int64) ([]Event, error) {
	var list []Event
	result := db.Order("time asc").
		Where("case_id = ?", cid).
		Find(&list)
	return list, result.Error
}

func GetEvent(cid int64, id int64) (Event, error) {
	x := Event{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveEvent(cid int64, x Event) (Event, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Time,
		x.AssetA,
		x.Direction,
		x.AssetB,
		x.Event,
		x.Raw,
		x.KeyEvent,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteEvent(cid int64, id int64) error {
	x := Event{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Assets
// --------------------------------------
func ListAssets(cid int64) ([]Asset, error) {
	var list []Asset
	result := db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func FindAssets(cid int64, term string) ([]Asset, error) {
	var list []Asset
	result := db.Order("name asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", term).
			Or("instr(name, ?) > 0", term).
			Or("instr(ip, ?) > 0", term).
			Or("instr(description, ?) > 0", term).
			Or("instr(compromised, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetAsset(cid int64, id int64) (Asset, error) {
	x := Asset{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveAsset(cid int64, x Asset) (Asset, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Type,
		x.Name,
		x.IP,
		x.Description,
		x.Compromised,
		x.Analysed,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteAsset(cid int64, id int64) error {
	x := Asset{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Malware
// --------------------------------------
func ListMalware(cid int64) ([]Malware, error) {
	var list []Malware
	result := db.Where("case_id = ?", cid).
		Order("filename asc").
		Find(&list)
	return list, result.Error
}

func FindMalware(cid int64, term string) ([]Malware, error) {
	var list []Malware
	result := db.Order("filename asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(filename, ?) > 0", term).
			Or("instr(filepath, ?) > 0", term).
			Or("instr(system, ?) > 0", term).
			Or("instr(hash, ?) > 0", term).
			Or("instr(notes, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetMalware(cid int64, id int64) (Malware, error) {
	x := Malware{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveMalware(cid int64, x Malware) (Malware, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Filename,
		x.Filepath,
		x.CDate,
		x.MDate,
		x.System,
		x.Hash,
		x.Notes,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteMalware(cid int64, id int64) error {
	x := Asset{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Indicators
// --------------------------------------
func ListIndicators(cid int64) ([]Indicator, error) {
	var list []Indicator
	result := db.
		Where("case_id = ?", cid).
		Order("type asc, value asc").
		Find(&list)
	return list, result.Error
}

func FindIndicators(cid int64, term string) ([]Indicator, error) {
	var list []Indicator
	result := db.Order("type asc, value asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", term).
			Or("instr(value, ?) > 0", term).
			Or("instr(description, ?) > 0", term).
			Or("instr(tlp, ?) > 0", term).
			Or("instr(source, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetIndicator(cid int64, id int64) (Indicator, error) {
	x := Indicator{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveIndicator(cid int64, x Indicator) (Indicator, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Type,
		x.Value,
		x.TLP,
		x.Description,
		x.Source,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteIndicator(cid int64, id int64) error {
	x := Indicator{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Users
// --------------------------------------
func ListUsers(cid int64) ([]User, error) {
	var list []User
	result := db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func FindUsers(cid int64, term string) ([]User, error) {
	var list []User
	result := db.Order("name asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(name, ?) > 0", term).
			Or("instr(company, ?) > 0", term).
			Or("instr(role, ?) > 0", term).
			Or("instr(email, ?) > 0", term).
			Or("instr(phone, ?) > 0", term).
			Or("instr(notes, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetUser(cid int64, id int64) (User, error) {
	x := User{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveUser(cid int64, x User) (User, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Name,
		x.Company,
		x.Role,
		x.Email,
		x.Phone,
		x.Notes,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteUser(cid int64, id int64) error {
	x := User{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Evidences
// --------------------------------------
func ListEvidences(cid int64) ([]Evidence, error) {
	var list []Evidence
	result := db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func FindEvidences(cid int64, term string) ([]Evidence, error) {
	var list []Evidence
	result := db.Order("name asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", term).
			Or("instr(name, ?) > 0", term).
			Or("instr(description, ?) > 0", term).
			Or("instr(hash, ?) > 0", term).
			Or("instr(location, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetEvidence(cid int64, id int64) (Evidence, error) {
	x := Evidence{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveEvidence(cid int64, x Evidence) (Evidence, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Type,
		x.Name,
		x.Description,
		x.Size,
		x.Hash,
		x.Location,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteEvidence(cid int64, id int64) error {
	x := Evidence{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Tasks
// --------------------------------------
func ListTasks(cid int64) ([]Task, error) {
	var list []Task
	result := db.
		Where("case_id = ?", cid).
		Order("date_due asc, type asc, task asc").
		Find(&list)
	return list, result.Error
}

func FindTasks(cid int64, term string) ([]Task, error) {
	var list []Task
	result := db.Order("date_due asc, type asc, task asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", term).
			Or("instr(task, ?) > 0", term).
			Or("instr(owner, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetTask(cid int64, id int64) (Task, error) {
	x := Task{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveTask(cid int64, x Task) (Task, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Type,
		x.Task,
		x.Done,
		x.Owner,
		x.DateDue,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteTask(cid int64, id int64) error {
	x := Task{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}

// --------------------------------------
// Notes
// --------------------------------------
func ListNotes(cid int64) ([]Note, error) {
	var list []Note
	result := db.
		Where("case_id = ?", cid).
		Order("category asc, title asc").
		Find(&list)
	return list, result.Error
}

func FindNotes(cid int64, term string) ([]Note, error) {
	var list []Note
	result := db.Order("category asc, title asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(category, ?) > 0", term).
			Or("instr(title, ?) > 0", term).
			Or("instr(description, ?) > 0", term)).
		Find(&list)
	return list, result.Error
}

func GetNote(cid int64, id int64) (Note, error) {
	x := Note{}
	result := db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func SaveNote(cid int64, x Note) (Note, error) {
	x.CRC = HashFields(
		x.CaseID,
		x.Title,
		x.Category,
		x.Description,
	)

	result := db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func DeleteNote(cid int64, id int64) error {
	x := Note{}
	return db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
