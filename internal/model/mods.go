package model

import (
	"database/sql"

	"github.com/sprungknoedl/dagobert/internal/fp"
)

var ModStatus = []string{
	"Running",
	"Failed",
	"Success",
}

var HookTrigger = []string{
	"OnEvidenceAdded",
}

type Mod struct {
	Name        string
	Description string
}

type Run struct {
	CaseID      string
	EvidenceID  string
	Name        string
	Description string
	Status      string
	Error       string
	Token       string
}

type Hook struct {
	ID        string
	Trigger   string
	Name      string
	Mod       string
	Condition string
	Enabled   bool
}

func (store *Store) GetRuns(base []Mod, eid string) ([]Run, error) {
	query := `
	SELECT case_id, evidence_id, name, description, status, error, token
	FROM runs
	WHERE evidence_id = :eid`

	rows, err := store.DB.Query(query,
		sql.Named("eid", eid))
	if err != nil {
		return nil, err
	}

	list := []Run{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	m := fp.ToMap(list, func(obj Run) string { return obj.Name })
	return fp.Apply(base, func(obj Mod) Run {
		return Run{
			Name:        obj.Name,
			Description: obj.Description,
			Status:      m[obj.Name].Status,
			Error:       m[obj.Name].Error,
		}
	}), nil
}

func (store *Store) GetActiveRuns() ([]Run, error) {
	query := `
	SELECT case_id, evidence_id, name, description, status, error, token
	FROM runs
	WHERE status = :status`

	rows, err := store.DB.Query(query,
		sql.Named("status", "Running"))
	if err != nil {
		return nil, err
	}

	list := []Run{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (store *Store) GetStaleRuns(token string) ([]Run, error) {
	query := `
	SELECT case_id, evidence_id, name, description, status, error, token
	FROM runs
	WHERE token != :token
	AND status = :status`

	rows, err := store.DB.Query(query,
		sql.Named("status", "Running"),
		sql.Named("token", token))
	if err != nil {
		return nil, err
	}

	list := []Run{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (store *Store) SaveRun(obj Run) error {
	query := `
	REPLACE INTO runs (case_id, evidence_id, name, description, status, error, token)
	VALUES (:case_id, :evidence_id, :name, :description, :status, :error, :token)`

	_, err := store.DB.Exec(query,
		sql.Named("case_id", obj.CaseID),
		sql.Named("evidence_id", obj.EvidenceID),
		sql.Named("name", obj.Name),
		sql.Named("description", obj.Description),
		sql.Named("status", obj.Status),
		sql.Named("error", obj.Error),
		sql.Named("token", obj.Token))
	return err
}

func (store *Store) ListHooks() ([]Hook, error) {
	query := `
	SELECT id, trigger, name, mod, condition, enabled
	FROM hooks`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	list := []Hook{}
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetHook(id string) (Hook, error) {
	query := `
	SELECT id, trigger, name, mod, condition, enabled
	FROM hooks
	WHERE id = :id`

	rows, err := store.DB.Query(query,
		sql.Named("id", id))
	if err != nil {
		return Hook{}, err
	}

	obj := Hook{}
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveHook(obj Hook) error {
	query := `
	INSERT INTO hooks (id, trigger, name, mod, condition, enabled)
	VALUES (:id, :trigger, :name, :mod, :condition, :enabled)
	ON CONFLICT (id)
		DO UPDATE SET trigger=:trigger, name=:name, mod=:mod, condition=:condition, enabled=:enabled
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("trigger", obj.Trigger),
		sql.Named("name", obj.Name),
		sql.Named("mod", obj.Mod),
		sql.Named("condition", obj.Condition),
		sql.Named("enabled", obj.Enabled))
	return err
}

func (store *Store) DeleteHook(id string) error {
	query := `
	DELETE FROM hooks
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
