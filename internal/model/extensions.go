package model

import (
	"database/sql"

	"github.com/sprungknoedl/dagobert/internal/fp"
)

var ExtensionStatus = []string{
	"Running",
	"Timeout",
	"Failed",
	"Success",
}

type Extension struct {
	Name        string
	Description string
	Supports    func(Evidence) bool
	Run         func(Store, Evidence) error
}

type Run struct {
	EvidenceID  string
	Name        string
	Description string
	Status      string
	Error       string
	TTL         Time
}

func (store *Store) GetRuns(base []Extension, eid string) ([]Run, error) {
	query := `
	SELECT evidence_id, name, description, status, error, ttl
	FROM runs
	WHERE evidence_id = :eid`

	rows, err := store.db.Query(query,
		sql.Named("eid", eid))
	if err != nil {
		return nil, err
	}

	var list []Run
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	m := fp.ToMap(list, func(obj Run) string { return obj.Name })
	return fp.Apply(base, func(obj Extension) Run {
		return Run{
			Name:        obj.Name,
			Description: obj.Description,
			Status:      m[obj.Name].Status,
			Error:       m[obj.Name].Error,
		}
	}), nil
}

func (store *Store) SaveRun(eid string, obj Run) error {
	query := `
	REPLACE INTO runs (evidence_id, name, description, status, error, ttl)
	VALUES (:evidence_id, :name, :description, :status, :error, :ttl)`

	_, err := store.db.Exec(query,
		sql.Named("evidence_id", eid),
		sql.Named("name", obj.Name),
		sql.Named("description", obj.Description),
		sql.Named("status", obj.Status),
		sql.Named("error", obj.Error),
		sql.Named("ttl", obj.TTL))
	return err
}
