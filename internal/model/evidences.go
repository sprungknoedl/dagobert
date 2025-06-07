package model

type Evidence struct {
	ID     string
	Type   string
	Name   string
	Hash   string
	Size   int64
	Source string
	Notes  string
	CaseID string
}

func (store *Store) ListEvidences(cid string) ([]Evidence, error) {
	list := []Evidence{}
	tx := store.DB.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetEvidence(cid string, id string) (Evidence, error) {
	obj := Evidence{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveEvidence(cid string, obj Evidence) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteEvidence(cid string, id string) error {
	return store.DB.Delete(Evidence{}, "id = ?", id).Error
}
