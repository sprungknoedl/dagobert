package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.TaskStore = &Store{}

func (store *Store) ListTasks(cid ulid.ULID) ([]model.Task, error) {
	var list []model.Task
	result := store.db.
		Where("case_id = ?", cid).
		Order("date_due asc, type asc, task asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindTasks(cid ulid.ULID, search string, sort string) ([]model.Task, error) {
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

func (store *Store) GetTask(cid ulid.ULID, id ulid.ULID) (model.Task, error) {
	x := model.Task{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) SaveTask(cid ulid.ULID, x model.Task) (model.Task, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteTask(cid ulid.ULID, id ulid.ULID) error {
	x := model.Task{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
