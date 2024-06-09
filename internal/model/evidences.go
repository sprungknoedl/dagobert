package model

import (
	"database/sql"
)

var EvidenceTypes = FromEnv("VALUES_EVIDENCE_TYPES", []string{"File", "Logs", "Artifacts Collection", "System Image", "Memory Dump", "Other"})

type Evidence struct {
	ID          string
	Type        string
	Name        string
	Description string
	Size        int64
	Hash        string
	Location    string
	CaseID      string
}

func (store *Store) FindEvidences(cid string, search string, sort string) ([]Evidence, error) {
	query := `
	SELECT id, type, name, description, size, hash, location, case_id
	FROM evidences
	WHERE case_id = :cid AND (
		instr(type, :search) > 0 OR
		instr(name, :search) > 0 OR
		instr(hash, :search) > 0 OR
		instr(location, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'hash'      THEN hash END ASC,
		CASE WHEN :sort = '-hash'     THEN hash END DESC,
		CASE WHEN :sort = 'location'  THEN location END ASC,
		CASE WHEN :sort = '-location' THEN location END DESC,
		CASE WHEN :sort = 'desc'      THEN description END ASC,
		CASE WHEN :sort = '-desc'     THEN description END DESC,
		CASE WHEN :sort = 'type'      THEN type END ASC,
		CASE WHEN :sort = '-type'     THEN type END DESC,
		CASE WHEN :sort = '-name'     THEN name END DESC,
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
	SELECT id, type, name, description, size, hash, location, case_id
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
	REPLACE INTO evidences (id, type, name, description, size, hash, location, case_id)
	VALUES (NULLIF(:id, ''), :type, :name, :description, :size, :hash, :location, :cid)`

	_, err := store.db.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("type", obj.Type),
		sql.Named("name", obj.Name),
		sql.Named("description", obj.Description),
		sql.Named("size", obj.Size),
		sql.Named("hash", obj.Hash),
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
