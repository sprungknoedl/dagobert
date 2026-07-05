package model

type Evidence struct {
	ID       string
	Type     string
	Name     string
	Hash     string
	Size     int64
	Source   string
	Notes    string
	Password string
	CaseID   string
	StartsAt Time
	EndsAt   Time
	Custom   Custom `form:"-"`
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
	tx := store.DB.First(&obj, "id = ? AND case_id = ?", id, cid)
	return obj, tx.Error
}

func (store *Store) SaveEvidence(cid string, obj Evidence) error {
	obj.CaseID = cid
	if err := store.assertCaseOwnership(&Evidence{}, obj.ID, cid); err != nil {
		return err
	}
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteEvidence(cid string, id string) error {
	return store.Transaction(func(tx *Store) error {
		res := tx.DB.Delete(Evidence{}, "id = ? AND case_id = ?", id, cid)
		if res.Error != nil || res.RowsAffected == 0 {
			return res.Error
		}
		return tx.DeleteEnrichments("Evidence", id)
	})
}
