package sqlite

import (
	"github.com/sprungknoedl/dagobert/model"
	"gorm.io/gorm/clause"
)

var _ model.TaskStore = &Store{}

func (store *Store) ListTasks(cid int64) ([]model.Task, error) {
	var list []model.Task
	result := store.db.
		Where("case_id = ?", cid).
		Order("date_due asc, type asc, task asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindTasks(cid int64, search string, sort string) ([]model.Task, error) {
	var list []model.Task
	query := store.db.
		Where("case_id = ?", cid).
		Where(store.db.
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

func (store *Store) GetTask(cid int64, id int64) (model.Task, error) {
	x := model.Task{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func (store *Store) SaveTask(cid int64, x model.Task) (model.Task, error) {
	x.CRC = model.HashFields(
		x.CaseID,
		x.Type,
		x.Task,
		x.Done,
		x.Owner,
		x.DateDue,
	)

	result := store.db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func (store *Store) DeleteTask(cid int64, id int64) error {
	x := model.Task{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
