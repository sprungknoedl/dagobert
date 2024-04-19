package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.NoteStore = &Store{}

func (store *Store) ListNotes(cid ulid.ULID) ([]model.Note, error) {
	var list []model.Note
	result := store.db.
		Where("case_id = ?", cid).
		Order("category asc, title asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindNotes(cid ulid.ULID, search string, sort string) ([]model.Note, error) {
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

func (store *Store) GetNote(cid ulid.ULID, id ulid.ULID) (model.Note, error) {
	x := model.Note{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) SaveNote(cid ulid.ULID, x model.Note) (model.Note, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteNote(cid ulid.ULID, id ulid.ULID) error {
	x := model.Note{}
	return store.db.
		Delete(&x, "id = ?", id).Error
}
