package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.EventStore = &Store{}

func (store *Store) FindEvents(cid ulid.ULID, search string, sort string) ([]model.Event, error) {
	var list []model.Event
	query := store.db.
		Where("case_id = ?", cid).
		Where(store.db.
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

func (store *Store) ListEvents(cid ulid.ULID) ([]model.Event, error) {
	var list []model.Event
	result := store.db.Order("time asc").
		Where("case_id = ?", cid).
		Find(&list)
	return list, result.Error
}

func (store *Store) GetEvent(cid ulid.ULID, id ulid.ULID) (model.Event, error) {
	x := model.Event{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) SaveEvent(cid ulid.ULID, x model.Event) (model.Event, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteEvent(cid ulid.ULID, id ulid.ULID) error {
	x := model.Event{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
