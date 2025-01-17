package model

import (
	"database/sql"
)

type Note struct {
	ID          string
	Title       string
	Category    string
	Description string
	CaseID      string
}

func (store *Store) ListNotes(cid string) ([]Note, error) {
	query := `
	SELECT id, title, category, description, case_id
	FROM notes
	WHERE case_id = :cid
	ORDER BY category ASC`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid))
	if err != nil {
		return nil, err
	}

	var list []Note
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetNote(cid string, id string) (Note, error) {
	query := `
	SELECT id, title, category, description, case_id
	FROM notes
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Note{}, err
	}

	var obj Note
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveNote(cid string, obj Note) error {
	query := `
	INSERT INTO notes (id, title, category, description, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :title, :category, :description, :cid)
	ON CONFLICT (id)
		DO UPDATE SET title=:title, category=:category, description=:description
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("title", obj.Title),
		sql.Named("category", obj.Category),
		sql.Named("description", obj.Description))
	return err
}

func (store *Store) DeleteNote(cid string, id string) error {
	query := `
	DELETE FROM notes
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
