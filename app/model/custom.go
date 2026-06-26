package model

// CustomAttribute is one admin-defined extra field for a given artifact-like
// entity. The Label is used verbatim as the value-map key and the
// report-template accessor — there is no separate slug.
type CustomAttribute struct {
	ID      string `gorm:"primaryKey"`
	Entity  string
	Label   string
	Type    string
	Options Strings `gorm:"type:text"`
	Rank    int
}

func (store *Store) ListCustomAttributes() ([]CustomAttribute, error) {
	list := []CustomAttribute{}
	tx := store.DB.Order("entity, rank, label asc").Find(&list)
	return list, tx.Error
}

func (store *Store) GetCustomAttribute(id string) (CustomAttribute, error) {
	obj := CustomAttribute{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveCustomAttribute(obj CustomAttribute) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteCustomAttribute(id string) error {
	return store.DB.Delete(&CustomAttribute{}, "id = ?", id).Error
}
