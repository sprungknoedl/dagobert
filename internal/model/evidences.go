package model

import (
	"database/sql"
)

var EvidenceTypes = FromEnv("VALUES_EVIDENCE_TYPES", []string{"File", "Logs", "Artifacts Collection", "System Image", "Memory Dump", "Malware", "Other"})

type Evidence struct {
	ID       string
	Type     string
	Name     string
	Hash     string
	Size     int64
	Notes    string
	Location string
	CaseID   string
}

func (store *Store) FindEvidences(cid string, search string, sort string) ([]Evidence, error) {
	query := `
	SELECT id, type, name, hash, size, notes, location, case_id
	FROM evidences
	WHERE case_id = :cid AND (
		instr(type, :search) > 0 OR
		instr(name, :search) > 0 OR
		instr(hash, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'hash'         THEN hash END ASC,
		CASE WHEN :sort = '-hash'        THEN hash END DESC,
		CASE WHEN :sort = 'notes'  THEN notes END ASC,
		CASE WHEN :sort = '-notes' THEN notes END DESC,
		CASE WHEN :sort = 'type'         THEN type END ASC,
		CASE WHEN :sort = '-type'        THEN type END DESC,
		CASE WHEN :sort = '-name'        THEN name END DESC,
		name ASC`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("search", search),
		sql.Named("sort", sort))
	if err != nil {
		return nil, err
	}

	var list []Evidence
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetEvidence(cid string, id string) (Evidence, error) {
	query := `
	SELECT id, type, name, hash, size, notes, location, case_id
	FROM evidences
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Evidence{}, err
	}

	var obj Evidence
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveEvidence(cid string, obj Evidence) error {
	query := `
	REPLACE INTO evidences (id, type, name, hash, size, notes, location, case_id)
	VALUES (NULLIF(:id, ''), :type, :name, :hash, :size, :notes, :location, :cid)`

	_, err := store.db.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("type", obj.Type),
		sql.Named("name", obj.Name),
		sql.Named("hash", obj.Hash),
		sql.Named("size", obj.Size),
		sql.Named("notes", obj.Notes),
		sql.Named("location", obj.Location))
	return err
}

func (store *Store) DeleteEvidence(cid string, id string) error {
	query := `
	DELETE FROM evidences
	WHERE id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
