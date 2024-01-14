package model

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func InitDatabase(dburl string) {
	debugLog := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  false,         // Disable color
		},
	)

	var err error
	db, err = gorm.Open(sqlite.Open(dburl), &gorm.Config{Logger: debugLog})
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
func FindCases(search string, sort string) ([]Case, error) {
	var list []Case
	query := db.
		Where("instr(name, ?) > 0", search).
		Or("instr(classification, ?) > 0", search).
		Or("instr(severity, ?) > 0", search).
		Or("instr(outcome, ?) > 0", search).
		Or("instr(summary, ?) > 0", search)

	switch sort {
	case "outcome":
		query = query.Order("outcome asc, name asc")
	case "-outcome":
		query = query.Order("outcome desc, name asc")
	case "severity":
		query = query.Order("severity asc, name asc")
	case "-severity":
		query = query.Order("classification desc, name asc")
	case "closed":
		query = query.Order("closed asc, name asc")
	case "-closed":
		query = query.Order("closed desc, name asc")
	case "summary":
		query = query.Order("summary asc, name asc")
	case "-summary":
		query = query.Order("summary desc, name asc")
	case "classification":
		query = query.Order("classification asc, name asc")
	case "-classification":
		query = query.Order("classification desc, name asc")
	case "-name":
		query = query.Order("name desc")
	default: // case "name"
		query = query.Order("name asc")
	}

	result := query.Find(&list)
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
	tx := db.Begin()
	tx.Delete(&Asset{}, "case_id = ?", id)
	tx.Delete(&Event{}, "case_id = ?", id)
	tx.Delete(&Evidence{}, "case_id = ?", id)
	tx.Delete(&Indicator{}, "case_id = ?", id)
	tx.Delete(&Malware{}, "case_id = ?", id)
	tx.Delete(&Note{}, "case_id = ?", id)
	tx.Delete(&Task{}, "case_id = ?", id)
	tx.Delete(&User{}, "case_id = ?", id)
	tx.Delete(&Case{}, id)
	return tx.Commit().Error
}

// --------------------------------------
// Events
// --------------------------------------
func FindEvents(cid int64, search string, sort string) ([]Event, error) {
	var list []Event
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", search).
			Or("instr(asset_a, ?) > 0", search).
			Or("instr(asset_b, ?) > 0", search).
			Or("instr(event, ?) > 0", search))

	switch sort {
	case "type":
		query = query.Order("type asc, time asc")
	case "-type":
		query = query.Order("type desc, time asc")
	case "src":
		query = query.Order("asset_a asc, time asc")
	case "-src":
		query = query.Order("asset_a desc, time asc")
	case "dst":
		query = query.Order("asset_b asc, time asc")
	case "-dst":
		query = query.Order("asset_b desc, time asc")
	case "event":
		query = query.Order("event asc, time asc")
	case "-event":
		query = query.Order("event desc, time asc")
	case "-time":
		query = query.Order("time desc, asset_a asc")
	default: // case "time":
		query = query.Order("time asc, asset_a asc")
	}

	result := query.Find(&list)
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

func FindAssets(cid int64, search string, sort string) ([]Asset, error) {
	var list []Asset
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", search).
			Or("instr(name, ?) > 0", search).
			Or("instr(ip, ?) > 0", search).
			Or("instr(description, ?) > 0", search).
			Or("instr(compromised, ?) > 0", search))

	switch sort {
	case "analysed":
		query = query.Order("analysed asc, name asc")
	case "-analysed":
		query = query.Order("analysed desc, name asc")
	case "compromised":
		query = query.Order("compromised asc, name asc")
	case "-compromised":
		query = query.Order("compromised desc, name asc")
	case "desc":
		query = query.Order("description asc, name asc")
	case "-desc":
		query = query.Order("description desc, name asc")
	case "ip":
		query = query.Order("ip asc, name asc")
	case "-ip":
		query = query.Order("ip desc, name asc")
	case "type":
		query = query.Order("type asc, name asc")
	case "-type":
		query = query.Order("type desc, name asc")
	case "-name":
		query = query.Order("name desc")
	default: // case "name"
		query = query.Order("name asc")
	}

	result := query.Find(&list)
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

func FindMalware(cid int64, search string, sort string) ([]Malware, error) {
	var list []Malware
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(filename, ?) > 0", search).
			Or("instr(filepath, ?) > 0", search).
			Or("instr(system, ?) > 0", search).
			Or("instr(hash, ?) > 0", search).
			Or("instr(notes, ?) > 0", search))

	switch sort {
	case "notes":
		query = query.Order("notes asc, filename asc")
	case "-notes":
		query = query.Order("notes desc, filename asc")
	case "hash":
		query = query.Order("hash asc, filename asc")
	case "-hash":
		query = query.Order("hash desc, filename asc")
	case "system":
		query = query.Order("system asc, filename asc")
	case "-system":
		query = query.Order("system desc, filename asc")
	case "filepath":
		query = query.Order("filepath asc, filename asc")
	case "-filepath":
		query = query.Order("filepath desc, filename asc")
	case "-filename":
		query = query.Order("filename desc")
	default: // case "filename":
		query = query.Order("filename asc")
	}

	result := query.Find(&list)
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

func FindIndicators(cid int64, search string, sort string) ([]Indicator, error) {
	var list []Indicator
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", search).
			Or("instr(value, ?) > 0", search).
			Or("instr(description, ?) > 0", search).
			Or("instr(tlp, ?) > 0", search).
			Or("instr(source, ?) > 0", search))

	switch sort {
	case "description":
		query = query.Order("description asc, type desc, value asc")
	case "-description":
		query = query.Order("description desc, type desc, value asc")
	case "source":
		query = query.Order("source asc, type desc, value asc")
	case "-source":
		query = query.Order("source desc, type desc, value asc")
	case "tlp":
		query = query.Order("tlp desc, type desc, value asc")
	case "-tlp":
		query = query.Order("tlp desc, type desc, value asc")
	case "value":
		query = query.Order("value desc")
	case "-value":
		query = query.Order("value desc")
	case "-type":
		query = query.Order("type desc, value asc")
	default: // case "type":
		query = query.Order("type asc, value asc")
	}

	result := query.Find(&list)
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

func FindUsers(cid int64, search string, sort string) ([]User, error) {
	var list []User
	query := db.Order("name asc").
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(name, ?) > 0", search).
			Or("instr(company, ?) > 0", search).
			Or("instr(role, ?) > 0", search).
			Or("instr(email, ?) > 0", search).
			Or("instr(phone, ?) > 0", search).
			Or("instr(notes, ?) > 0", search))

	switch sort {
	case "notes":
		query = query.Order("notes asc, name asc")
	case "-notes":
		query = query.Order("notes desc, name asc")
	case "phone":
		query = query.Order("phone asc, name asc")
	case "-phone":
		query = query.Order("phone desc, name asc")
	case "email":
		query = query.Order("email asc, name asc")
	case "-email":
		query = query.Order("email desc, name asc")
	case "role":
		query = query.Order("role asc, name asc")
	case "-role":
		query = query.Order("role desc, name asc")
	case "company":
		query = query.Order("company asc, name asc")
	case "-company":
		query = query.Order("company desc, name asc")
	case "-name":
		query = query.Order("name desc")
	default: // case "name"
		query = query.Order("name asc")
	}

	result := query.Find(&list)
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

func FindEvidences(cid int64, search string, sort string) ([]Evidence, error) {
	var list []Evidence
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", search).
			Or("instr(name, ?) > 0", search).
			Or("instr(description, ?) > 0", search).
			Or("instr(hash, ?) > 0", search).
			Or("instr(location, ?) > 0", search))

	switch sort {
	case "location":
		query = query.Order("location asc, name asc")
	case "-location":
		query = query.Order("location desc, name asc")
	case "hash":
		query = query.Order("hash asc, name asc")
	case "-hash":
		query = query.Order("hash desc, name asc")
	case "description":
		query = query.Order("description asc, name asc")
	case "-description":
		query = query.Order("description desc, name asc")
	case "type":
		query = query.Order("type asc, name asc")
	case "-type":
		query = query.Order("type desc, name asc")
	case "-name":
		query = query.Order("name desc")
	default: // case "name":
		query = query.Order("name asc")
	}

	result := query.Find(&list)
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

func FindTasks(cid int64, search string, sort string) ([]Task, error) {
	var list []Task
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(type, ?) > 0", search).
			Or("instr(task, ?) > 0", search).
			Or("instr(owner, ?) > 0", search))

	switch sort {
	case "type":
		query = query.Order("type asc, date_due asc")
	case "-type":
		query = query.Order("type desc, date_due asc")
	case "task":
		query = query.Order("task desc, date_due asc")
	case "-task":
		query = query.Order("task asc, date_due asc")
	case "owner":
		query = query.Order("owner asc, date_due asc")
	case "-owner":
		query = query.Order("owner desc, date_due asc")
	case "-due":
		query = query.Order("date_due desc, type asc, task asc")
	default: // case "due"
		query = query.Order("date_due asc, type asc, task asc")
	}

	result := query.Find(&list)
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

func FindNotes(cid int64, search string, sort string) ([]Note, error) {
	var list []Note
	query := db.
		Where("case_id = ?", cid).
		Where(db.
			Where("instr(category, ?) > 0", search).
			Or("instr(title, ?) > 0", search).
			Or("instr(description, ?) > 0", search))

	switch sort {
	case "title":
		query = query.Order("title asc")
	case "-title":
		query = query.Order("title desc")
	case "desc":
		query = query.Order("description asc")
	case "-desc":
		query = query.Order("description desc")
	case "-category":
		query = query.Order("category desc, title asc")
	default: // case "category"
		query = query.Order("category asc, title asc")
	}

	result := query.Find(&list)
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
