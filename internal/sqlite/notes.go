package sqlite

import (
	"github.com/sprungknoedl/dagobert/model"
	"gorm.io/gorm/clause"
)

var _ model.NoteStore = &Store{}

func (store *Store) ListNotes(cid int64) ([]model.Note, error) {
	var list []model.Note
	result := store.db.
		Where("case_id = ?", cid).
		Order("category asc, title asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindNotes(cid int64, search string, sort string) ([]model.Note, error) {
	var list []model.Note
	query := store.db.
		Where("case_id = ?", cid).
		Where(store.db.
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

func (store *Store) GetNote(cid int64, id int64) (model.Note, error) {
	x := model.Note{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func (store *Store) SaveNote(cid int64, x model.Note) (model.Note, error) {
	x.CRC = model.HashFields(
		x.CaseID,
		x.Title,
		x.Category,
		x.Description,
	)

	result := store.db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func (store *Store) DeleteNote(cid int64, id int64) error {
	x := model.Note{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
