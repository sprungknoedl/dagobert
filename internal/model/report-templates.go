package model

type ReportTemplate struct {
	ID    string
	Name  string
	Notes string
}

func (store *Store) ListReportTemplates() ([]ReportTemplate, error) {
	list := []ReportTemplate{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetReportTemplate(id string) (ReportTemplate, error) {
	obj := ReportTemplate{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) GetReportTemplateByName(name string) (ReportTemplate, error) {
	obj := ReportTemplate{}
	tx := store.DB.First(&obj, "name = ?", name)
	return obj, tx.Error
}

func (store *Store) SaveReportTemplate(obj ReportTemplate) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteReportTemplate(id string) error {
	return store.DB.Delete(&ReportTemplate{}, "id = ?", id).Error
}
