package model

import (
	"database/sql"
)

var CaseSeverities = FromEnv("VALUES_CASE_SEVERITIES", []string{"", "Low", "Medium", "High"})
var CaseOutcomes = FromEnv("VALUES_CASE_OUTCOMES", []string{"", "False positive", "True positive", "Benign positive"})

type Case struct {
	ID             string
	Name           string
	Summary        string
	Classification string
	Severity       string
	Outcome        string
	Closed         bool

	Assets     []Asset
	Evidences  []Evidence
	Indicators []Indicator
	Events     []Event
	Malware    []Malware
	Notes      []Note
	Tasks      []Task
}

func (store *Store) ListCases() ([]Case, error) {
	query := `
	SELECT id, name, summary, classification, severity, outcome, closed
	FROM cases
	ORDER BY name ASC`

	rows, err := store.db.Query(query)
	if err != nil {
		return nil, err
	}

	var list []Case
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetCase(cid string) (Case, error) {
	query := `
	SELECT id, name, summary, classification, severity, outcome, closed
	FROM cases
	WHERE id = :cid`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid))
	if err != nil {
		return Case{}, err
	}

	var obj Case
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) GetCaseFull(cid string) (Case, error) {
	obj, err := store.GetCase(cid)
	if err != nil {
		return Case{}, err
	}

	// TODO: fetch relations

	return obj, nil
}

func (store *Store) SaveCase(obj Case) error {
	query := `
	INSERT INTO cases (id, name, closed, classification, severity, outcome, summary)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :name, :closed, :classification, :severity, :outcome, :summary)
	ON CONFLICT (id)
		DO UPDATE SET name=:name, closed=:closed, classification=:classification, severity=:severity, outcome=:outcome, summary=:summary
		WHERE id = :id
	`

	_, err := store.db.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("name", obj.Name),
		sql.Named("closed", obj.Closed),
		sql.Named("classification", obj.Classification),
		sql.Named("severity", obj.Severity),
		sql.Named("outcome", obj.Outcome),
		sql.Named("summary", obj.Summary))
	return err
}

func (store *Store) DeleteCase(cid string) error {
	query := `
	DELETE FROM cases
	WHERE id = :cid`

	_, err := store.db.Exec(query,
		sql.Named("cid", cid))
	return err
}
