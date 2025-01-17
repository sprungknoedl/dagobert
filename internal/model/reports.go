package model

import (
	"database/sql"
)

type Report struct {
	ID    string
	Name  string
	Notes string
}

func (store *Store) ListReports() ([]Report, error) {
	query := `
	SELECT id, name, notes
	FROM reports
	ORDER BY name ASC`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	var list []Report
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetReport(id string) (Report, error) {
	query := `
	SELECT id, name, notes
	FROM reports
	WHERE id = :id
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("id", id))
	if err != nil {
		return Report{}, err
	}

	var obj Report
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) GetReportByName(name string) (Report, error) {
	query := `
	SELECT id, name, notes
	FROM reports
	WHERE name = :name
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("name", name))
	if err != nil {
		return Report{}, err
	}

	var obj Report
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveReport(obj Report) error {
	query := `
	INSERT INTO reports (id, name, notes)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :name, :notes)
	ON CONFLICT (id)
		DO UPDATE SET name=:name, notes=:notes
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("name", obj.Name),
		sql.Named("notes", obj.Notes))
	return err
}

func (store *Store) DeleteReport(id string) error {
	query := `
	DELETE FROM reports
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
