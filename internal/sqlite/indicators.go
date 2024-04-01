package sqlite

import (
	"github.com/sprungknoedl/dagobert/pkg/model"
	"gorm.io/gorm/clause"
)

var _ model.IndicatorStore = &Store{}

func (store *Store) ListIndicators(cid int64) ([]model.Indicator, error) {
	var list []model.Indicator
	result := store.db.
		Where("case_id = ?", cid).
		Order("type asc, value asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindIndicators(cid int64, search string, sort string) ([]model.Indicator, error) {
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

func (store *Store) GetIndicator(cid int64, id int64) (model.Indicator, error) {
	x := model.Indicator{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func (store *Store) SaveIndicator(cid int64, x model.Indicator) (model.Indicator, error) {
	x.CRC = model.HashFields(
		x.CaseID,
		x.Type,
		x.Value,
		x.TLP,
		x.Description,
		x.Source,
	)

	result := store.db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func (store *Store) DeleteIndicator(cid int64, id int64) error {
	x := model.Indicator{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
