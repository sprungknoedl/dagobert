package model

import "github.com/sprungknoedl/dagobert/pkg/fp"

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
	store.customAttributesMu.Lock()
	defer store.customAttributesMu.Unlock()
	if store.customAttributesCache != nil {
		return *store.customAttributesCache, nil
	}

	list := []CustomAttribute{}
	tx := store.DB.Order("entity, rank, label asc").Find(&list)
	if tx.Error != nil {
		return nil, tx.Error
	}

	store.customAttributesCache = &list
	return list, nil
}

func (store *Store) GetCustomAttribute(id string) (CustomAttribute, error) {
	obj := CustomAttribute{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveCustomAttribute(obj CustomAttribute) error {
	err := store.DB.Save(obj).Error
	store.customAttributesMu.Lock()
	store.customAttributesCache = nil
	store.customAttributesMu.Unlock()
	return err
}

func (store *Store) DeleteCustomAttribute(id string) error {
	err := store.DB.Delete(&CustomAttribute{}, "id = ?", id).Error
	store.customAttributesMu.Lock()
	store.customAttributesCache = nil
	store.customAttributesMu.Unlock()
	return err
}

// EnsureCustomAttribute creates the (entity, label) definition if it does not
// already exist. It is idempotent and never touches an existing row, so admin
// tweaks to Rank/Options/Type survive. Enrichment modules call it at worker
// startup so their attributes are recreated if an admin deletes one.
func (store *Store) EnsureCustomAttribute(entity, label, typ string, options Strings, rank int) error {
	var count int64
	err := store.DB.Model(&CustomAttribute{}).
		Where("entity = ? AND label = ?", entity, label).
		Count(&count).Error
	if err != nil || count > 0 {
		return err
	}

	err = store.DB.Create(&CustomAttribute{
		ID:      fp.Random(10),
		Entity:  entity,
		Label:   label,
		Type:    typ,
		Options: options,
		Rank:    rank,
	}).Error

	store.customAttributesMu.Lock()
	store.customAttributesCache = nil
	store.customAttributesMu.Unlock()
	return err
}
