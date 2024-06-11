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

func (store *Store) FindNotes(cid string, search string, sort string) ([]Note, error) {
	query := `
	SELECT id, title, category, description, case_id
	FROM notes
	WHERE case_id = :cid AND (
		instr(category, :search) > 0 OR
		instr(title, :search) > 0 OR
		instr(description, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'title'        THEN title END ASC,
		CASE WHEN :sort = '-title'       THEN title END DESC,
		CASE WHEN :sort = 'description'  THEN description END ASC,
		CASE WHEN :sort = '-description' THEN description END DESC,
		CASE WHEN :sort = '-category'    THEN category END DESC,
		category ASC`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("search", search),
		sql.Named("sort", sort))
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

	rows, err := store.db.Query(query,
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
	REPLACE INTO notes (id, title, category, description, case_id)
	VALUES (NULLIF(:id, ''), :title, :category, :description, :cid)`

	_, err := store.db.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("title", obj.Title),
		sql.Named("category", obj.Category),
		sql.Named("description", obj.Description))
	return err
}

func (store *Store) DeleteNote(cid string, id string) error {
	query := `
	DELETE FROM malware
	WHERE id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
