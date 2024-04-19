package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.EvidenceStore = &Store{}

func (store *Store) ListEvidences(cid ulid.ULID) ([]model.Evidence, error) {
	var list []model.Evidence
	result := store.db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindEvidences(cid ulid.ULID, search string, sort string) ([]model.Evidence, error) {
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

func (store *Store) GetEvidence(cid ulid.ULID, id ulid.ULID) (model.Evidence, error) {
	x := model.Evidence{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) SaveEvidence(cid ulid.ULID, x model.Evidence) (model.Evidence, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteEvidence(cid ulid.ULID, id ulid.ULID) error {
	x := model.Evidence{}
	return store.db.
		Delete(&x, "id = ?", id).Error
}
