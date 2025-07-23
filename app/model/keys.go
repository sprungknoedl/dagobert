package model

type Key struct {
	Key  string `gorm:"primaryKey"`
	Name string
	Type string
}

func (store *Store) ListKeys() ([]Key, error) {
	list := []Key{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetKey(key string) (Key, error) {
	obj := Key{}
	tx := store.DB.First(&obj, "key = ?", key)
	return obj, tx.Error
}

func (store *Store) SaveKey(obj Key) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteKey(key string) error {
	return store.DB.Delete(Key{}, "key = ?", key).Error
}
