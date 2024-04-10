package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.IndicatorStore = &Store{}

func (store *Store) ListIndicators(cid ulid.ULID) ([]model.Indicator, error) {
	var list []model.Indicator
	result := store.db.
		Where("case_id = ?", cid).
		Order("type asc, value asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindIndicators(cid ulid.ULID, search string, sort string) ([]model.Indicator, error) {
	var list []model.Indicator
	query := store.db.
		Where("case_id = ?", cid).
		Where(store.db.
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

func (store *Store) GetIndicator(cid ulid.ULID, id ulid.ULID) (model.Indicator, error) {
	x := model.Indicator{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) SaveIndicator(cid ulid.ULID, x model.Indicator) (model.Indicator, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteIndicator(cid ulid.ULID, id ulid.ULID) error {
	x := model.Indicator{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
