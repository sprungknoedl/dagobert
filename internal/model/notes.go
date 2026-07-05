package model

type Note struct {
	ID          string
	Title       string
	Category    string
	Description string
	CaseID      string
	Custom      Custom `form:"-"`
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
	tx := store.DB.First(&obj, "id = ? AND case_id = ?", id, cid)
	return obj, tx.Error
}

func (store *Store) SaveNote(cid string, obj Note) error {
	obj.CaseID = cid
	if err := store.assertCaseOwnership(&Note{}, obj.ID, cid); err != nil {
		return err
	}
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteNote(cid string, id string) error {
	return store.DB.Delete(&Note{}, "id = ? AND case_id = ?", id, cid).Error
}
