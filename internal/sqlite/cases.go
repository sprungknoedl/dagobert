package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
	"gorm.io/gorm/clause"
)

var _ model.CaseStore = &Store{}

func (store *Store) FindCases(search string, sort string) ([]model.Case, error) {
	var list []model.Case
	query := store.db.
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

func (store *Store) ListCases() ([]model.Case, error) {
	var list []model.Case
	result := store.db.Order("name asc").Find(&list)
	return list, result.Error
}

func (store *Store) GetCase(id ulid.ULID) (model.Case, error) {
	x := model.Case{}
	result := store.db.First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) GetCaseFull(id ulid.ULID) (model.Case, error) {
	x := model.Case{}
	result := store.db.
		Preload(clause.Associations).
		First(&x, id)
	return x, result.Error
}

func (store *Store) SaveCase(x model.Case) (model.Case, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteCase(id ulid.ULID) error {
	tx := store.db.Begin()
	tx.Delete(&model.Asset{}, "case_id = ?", id)
	tx.Delete(&model.Event{}, "case_id = ?", id)
	tx.Delete(&model.Evidence{}, "case_id = ?", id)
	tx.Delete(&model.Indicator{}, "case_id = ?", id)
	tx.Delete(&model.Malware{}, "case_id = ?", id)
	tx.Delete(&model.Note{}, "case_id = ?", id)
	tx.Delete(&model.Task{}, "case_id = ?", id)
	tx.Delete(&model.Case{}, "id = ?", id)
	return tx.Commit().Error
}
