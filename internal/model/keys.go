package model

import (
	"database/sql"
)

type Key struct {
	Key  string
	Name string
}

func (store *Store) FindKeys(search string, sort string) ([]Key, error) {
	query := `
	SELECT
		key, name
	FROM 
		keys
	WHERE 
		instr(name, :search) > 0
	ORDER BY
		name`

	rows, err := store.db.Query(query,
		sql.Named("search", search),
		sql.Named("sort", sort))
	if err != nil {
		return nil, err
	}

	var list []Key
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetKey(key string) (Key, error) {
	query := `
	SELECT
		key, name
	FROM
		keys
	WHERE
		key = :key`

	rows, err := store.db.Query(query,
		sql.Named("key", key))
	if err != nil {
		return Key{}, err
	}

	var obj Key
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveKey(obj Key) error {
	query := `
	INSERT INTO keys (key, name)
	VALUES (NULLIF(:key, ''), :name)
	ON CONFLICT (key)
		DO UPDATE SET name=:name
		WHERE key = :key`

	_, err := store.db.Exec(query,
		sql.Named("key", obj.Key),
		sql.Named("name", obj.Name))
	return err
}

func (store *Store) DeleteKey(id string) error {
	query := `
	DELETE FROM keys
	WHERE key = :key`

	_, err := store.db.Exec(query,
		sql.Named("key", id))
	return err
}
