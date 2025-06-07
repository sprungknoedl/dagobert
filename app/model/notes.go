package model

type Note struct {
	ID          string
	Title       string
	Category    string
	Description string
	CaseID      string
}

func (store *Store) ListNotes(cid string) ([]Note, error) {
	list := []Note{}
	tx := store.DB.
		Where("case_id = ?", cid).
		Order("category asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetNote(cid string, id string) (Note, error) {
	obj := Note{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveNote(cid string, obj Note) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteNote(cid string, id string) error {
	return store.DB.Delete(&Note{}, "id = ?", id).Error
}
