package model

type Report struct {
	ID    string
	Name  string
	Notes string
}

func (store *Store) ListReports() ([]Report, error) {
	list := []Report{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetReport(id string) (Report, error) {
	obj := Report{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) GetReportByName(name string) (Report, error) {
	obj := Report{}
	tx := store.DB.First(&obj, "name = ?", name)
	return obj, tx.Error
}

func (store *Store) SaveReport(obj Report) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteReport(id string) error {
	return store.DB.Delete(&Report{}, "id = ?", id).Error
}
