package model

import (
	"database/sql"
)

var KeyTypes = []string{"API", "Dagobert", "Donald"}

type Key struct {
	Type string
	Key  string
	Name string
}

func (store *Store) ListKeys() ([]Key, error) {
	query := `
	SELECT type, key, name
	FROM keys
	ORDER BY name`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	list := []Key{}
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetKey(key string) (Key, error) {
	query := `
	SELECT type, key, name
	FROM keys
	WHERE key = :key`

	rows, err := store.DB.Query(query,
		sql.Named("key", key))
	if err != nil {
		return Key{}, err
	}

	obj := Key{}
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveKey(obj Key) error {
	query := `
	INSERT INTO keys (type, key, name)
	VALUES (:type, :key, :name)
	ON CONFLICT (key)
		DO UPDATE SET type=:type, name=:name
		WHERE key = :key`

	_, err := store.DB.Exec(query,
		sql.Named("type", obj.Type),
		sql.Named("key", obj.Key),
		sql.Named("name", obj.Name))
	return err
}

func (store *Store) DeleteKey(id string) error {
	query := `
	DELETE FROM keys
	WHERE key = :key`

	_, err := store.DB.Exec(query,
		sql.Named("key", id))
	return err
}
