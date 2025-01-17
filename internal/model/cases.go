package model

import (
	"database/sql"
	"fmt"
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

	SketchID int

	Assets     []Asset
	Evidences  []Evidence
	Indicators []Indicator
	Events     []Event
	Malware    []Malware
	Notes      []Note
	Tasks      []Task
}

func (c Case) String() string {
	if c.ID != "" {
		return fmt.Sprintf("#%s - %s", c.ID, c.Name)
	} else {
		return ""
	}
}

func (store *Store) ListCases() ([]Case, error) {
	query := `
	SELECT id, name, summary, classification, severity, outcome, closed, sketch_id
	FROM cases
	ORDER BY name ASC`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	var list []Case
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetCase(cid string) (Case, error) {
	query := `
	SELECT id, name, summary, classification, severity, outcome, closed, sketch_id
	FROM cases
	WHERE id = :cid`

	rows, err := store.DB.Query(query,
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
	INSERT INTO cases (id, name, summary, classification, severity, outcome, closed, sketch_id)
	VALUES (:id, :name, :summary, :classification, :severity, :outcome, :closed, :sketch_id)
	ON CONFLICT (id)
		DO UPDATE SET name=:name, summary=:summary, classification=:classification, severity=:severity, outcome=:outcome, closed=:closed, sketch_id=:sketch_id
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("name", obj.Name),
		sql.Named("summary", obj.Summary),
		sql.Named("classification", obj.Classification),
		sql.Named("severity", obj.Severity),
		sql.Named("outcome", obj.Outcome),
		sql.Named("closed", obj.Closed),
		sql.Named("sketch_id", obj.SketchID))
	return err
}

func (store *Store) DeleteCase(cid string) error {
	query := `
	DELETE FROM cases
	WHERE id = :cid`

	_, err := store.DB.Exec(query,
		sql.Named("cid", cid))
	return err
}
