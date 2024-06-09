package model

import (
	"database/sql"
)

var IndicatorTypes = FromEnv("VALUES_INDICATOR_TYPES", []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"})
var IndicatorTLPs = FromEnv("VALUES_INDICATOR_TLPS", []string{"TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"})

type Indicator struct {
	ID          string
	Type        string
	Value       string
	TLP         string
	Description string
	Source      string
	CaseID      string
}

func (store *Store) FindIndicators(cid string, search string, sort string) ([]Indicator, error) {
	query := `
	SELECT id, type, value, tlp, description, source, case_id
	FROM indicators
	WHERE case_id = :cid AND (
		instr(type, :search) > 0 OR
		instr(value, :search) > 0 OR
		instr(source, :search) > 0 OR
		instr(description, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'source'  THEN source END ASC,
		CASE WHEN :sort = '-source' THEN source END DESC,
		CASE WHEN :sort = 'tlp'     THEN tlp END ASC,
		CASE WHEN :sort = '-tlp'    THEN tlp END DESC,
		CASE WHEN :sort = 'desc'    THEN description END ASC,
		CASE WHEN :sort = '-desc'   THEN description END DESC,
		CASE WHEN :sort = 'type'    THEN type END ASC,
		CASE WHEN :sort = '-type'   THEN type END DESC,
		CASE WHEN :sort = '-value'  THEN value END DESC,
		value ASC`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("search", search),
		sql.Named("sort", sort))
	if err != nil {
		return nil, err
	}

	var list []Indicator
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetIndicator(cid string, id string) (Indicator, error) {
	query := `
	SELECT id, type, value, tlp, description, source, case_id
	FROM indicators
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Indicator{}, err
	}

	var obj Indicator
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveIndicator(cid string, obj Indicator) error {
	query := `
	REPLACE INTO indicators (id, type, value, tlp, description, source, case_id)
	VALUES (NULLIF(:id, ''), :type, :value, :tlp, :description, :source, :cid)`

	_, err := store.db.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("type", obj.Type),
		sql.Named("value", obj.Value),
		sql.Named("tlp", obj.TLP),
		sql.Named("description", obj.Description),
		sql.Named("source", obj.Source))
	return err
}

func (store *Store) DeleteIndicator(cid string, id string) error {
	query := `
	DELETE FROM indicators
	WHERE id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
