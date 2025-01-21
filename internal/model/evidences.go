package model

import (
	"database/sql"
)

var EvidenceTypes = FromEnv("VALUES_EVIDENCE_TYPES", []string{"File", "Logs", "Triage", "System Image", "Memory Dump", "Malware", "Other"})

type Evidence struct {
	ID       string
	Type     string
	Name     string
	Hash     string
	Size     int64
	Source   string
	Notes    string
	Location string
	CaseID   string
}

func (store *Store) ListEvidences(cid string) ([]Evidence, error) {
	query := `
	SELECT id, type, name, hash, size, source, notes, location, case_id
	FROM evidences
	WHERE case_id = :cid
	ORDER BY name ASC`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid))
	if err != nil {
		return nil, err
	}

	list := []Evidence{}
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetEvidence(cid string, id string) (Evidence, error) {
	query := `
	SELECT id, type, name, hash, size, source, notes, location, case_id
	FROM evidences
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Evidence{}, err
	}

	obj := Evidence{}
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveEvidence(cid string, obj Evidence) error {
	query := `
	INSERT INTO evidences (id, type, name, hash, size, source, notes, location, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :type, :name, :hash, :size, :source, :notes, :location, :cid)
	ON CONFLICT (id)
		DO UPDATE SET type=:type, name=:name, source=:source, notes=:notes
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("type", obj.Type),
		sql.Named("name", obj.Name),
		sql.Named("hash", obj.Hash),
		sql.Named("size", obj.Size),
		sql.Named("source", obj.Source),
		sql.Named("notes", obj.Notes),
		sql.Named("location", obj.Location))
	return err
}

func (store *Store) DeleteEvidence(cid string, id string) error {
	query := `
	DELETE FROM evidences
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
