package sqlite

import (
	"github.com/sprungknoedl/dagobert/pkg/model"
	"gorm.io/gorm/clause"
)

var _ model.EvidenceStore = &Store{}

func (store *Store) ListEvidences(cid int64) ([]model.Evidence, error) {
	var list []model.Evidence
	result := store.db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindEvidences(cid int64, search string, sort string) ([]model.Evidence, error) {
	var list []model.Evidence
	query := store.db.
		Where("case_id = ?", cid).
		Where(store.db.
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

func (store *Store) GetEvidence(cid int64, id int64) (model.Evidence, error) {
	x := model.Evidence{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func (store *Store) SaveEvidence(cid int64, x model.Evidence) (model.Evidence, error) {
	x.CRC = model.HashFields(
		x.CaseID,
		x.Type,
		x.Name,
		x.Description,
		x.Size,
		x.Hash,
		x.Location,
	)

	result := store.db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func (store *Store) DeleteEvidence(cid int64, id int64) error {
	x := model.Evidence{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
