package model

import (
	"database/sql"
	"time"
)

var IndicatorStatus = FromEnv("VALUES_INDICATOR_STATUS", []string{"Confirmed", "Suspicious", "Under investigation", "Unrelated"})
var IndicatorTypes = FromEnv("VALUES_INDICATOR_TYPES", []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"})
var IndicatorTLPs = FromEnv("VALUES_INDICATOR_TLPS", []string{"TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"})

type Indicator struct {
	ID     string
	Status string
	Type   string
	Value  string
	TLP    string
	Source string
	Notes  string
	CaseID string

	RawFirstSeen string
	FirstSeen    time.Time
}

func (store *Store) FindIndicators(cid string, search string, sort string) ([]Indicator, error) {
	query := `
	SELECT 
		i.id, i.status, i.type, i.value, i.tlp, i.source, i.notes, i.case_id,
		IFNULL(min(e.time), '')
	FROM
		indicators i
	LEFT OUTER JOIN
		event_indicators ei ON i.id = ei.indicator_id 
	LEFT OUTER JOIN
		events e ON e.id = ei.event_id 
	WHERE i.case_id = :cid AND (
		instr(i.status, :search) > 0 OR
		instr(i.type, :search) > 0 OR
		instr(i.value, :search) > 0 OR
		instr(i.source, :search) > 0 OR
		instr(i.notes, :search) > 0)
	GROUP BY 
		i.id, i.type, i.value, i.tlp, i.notes, i.source, i.case_id
	ORDER BY
		CASE WHEN :sort = 'source'  THEN i.source END ASC,
		CASE WHEN :sort = '-source' THEN i.source END DESC,
		CASE WHEN :sort = 'tlp'     THEN i.tlp END ASC,
		CASE WHEN :sort = '-tlp'    THEN i.tlp END DESC,
		CASE WHEN :sort = 'notes'   THEN i.notes END ASC,
		CASE WHEN :sort = '-notes'  THEN i.notes END DESC,
		CASE WHEN :sort = 'type'    THEN i.type END ASC,
		CASE WHEN :sort = '-type'   THEN i.type END DESC,
		CASE WHEN :sort = 'status'  THEN i.status END ASC,
		CASE WHEN :sort = '-status' THEN i.status END DESC,
		CASE WHEN :sort = 'first seen'    THEN 7 END ASC,
		CASE WHEN :sort = '-first seen'   THEN 7 END DESC,
		CASE WHEN :sort = '-value'  THEN i.value END DESC,
		i.value ASC`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("search", search),
		sql.Named("sort", sort))
	if err != nil {
		return nil, err
	}

	var list []Indicator
	err = ScanAll(rows, &list)

	// transform raw relations / fields
	for i, elem := range list {
		elem.FirstSeen, _ = time.Parse("2006-01-02 15:04:05Z07:00", elem.RawFirstSeen)
		list[i] = elem
	}

	return list, err
}

func (store *Store) GetIndicator(cid string, id string) (Indicator, error) {
	query := `
	SELECT id, status, type, value, tlp, source, notes, case_id
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

func (store *Store) GetIndicatorByValue(cid string, value string) (Indicator, error) {
	query := `
	SELECT id, status, type, value, tlp, source, notes, case_id
	FROM indicators
	WHERE case_id = :cid AND value = :value
	LIMIT 1`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("value", value))
	if err != nil {
		return Indicator{}, err
	}

	var obj Indicator
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveIndicator(cid string, obj Indicator) (Indicator, error) {
	query := `
	INSERT INTO indicators (id, status, type, value, tlp, source, notes, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :status, :type, :value, :tlp, :source, :notes, :cid)
	ON CONFLICT (id)
		DO UPDATE SET status=:status, type=:type, value=:value, tlp=:tlp, source=:source, notes=:notes
		WHERE id = :id
	RETURNING id, status, type, value, tlp, source, notes, case_id`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("status", obj.Status),
		sql.Named("type", obj.Type),
		sql.Named("value", obj.Value),
		sql.Named("tlp", obj.TLP),
		sql.Named("source", obj.Source),
		sql.Named("notes", obj.Notes))
	if err != nil {
		return Indicator{}, err
	}

	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) DeleteIndicator(cid string, id string) error {
	query := `
	DELETE FROM indicators
	WHERE id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
