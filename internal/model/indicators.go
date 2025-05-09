package model

import (
	"database/sql"
)

type Indicator struct {
	ID     string
	Status string
	Type   string
	Value  string
	TLP    string
	Source string
	Notes  string
	CaseID string

	FirstSeen Time
	LastSeen  Time
	Events    int
}

func (store *Store) ListIndicators(cid string) ([]Indicator, error) {
	query := `
	SELECT 
		i.id, i.status, i.type, i.value, i.tlp, i.source, i.notes, i.case_id,
		(SELECT  min(e.time)
			FROM events e
			LEFT JOIN event_indicators ON e.id = event_indicators.event_id 
			WHERE event_indicators.indicator_id = i.id) AS first_seen,
		(SELECT  max(e.time)
			FROM events e
			LEFT JOIN event_indicators ON e.id = event_indicators.event_id 
			WHERE event_indicators.indicator_id = i.id) AS last_seen,
		(SELECT  count()
			FROM events e
			LEFT JOIN event_indicators ON e.id = event_indicators.event_id 
			WHERE event_indicators.indicator_id = i.id) AS events
	FROM indicators i
	WHERE case_id = :cid
	ORDER BY i.type ASC, i.value ASC`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid))
	if err != nil {
		return nil, err
	}

	list := []Indicator{}
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetIndicator(cid string, id string) (Indicator, error) {
	query := `
	SELECT id, status, type, value, tlp, source, notes, case_id,
		(SELECT  min(e.time)
			FROM events e
			LEFT JOIN event_indicators ON e.id = event_indicators.event_id 
			WHERE event_indicators.indicator_id = i.id) AS first_seen,
		(SELECT  max(e.time)
			FROM events e
			LEFT JOIN event_indicators ON e.id = event_indicators.event_id 
			WHERE event_indicators.indicator_id = i.id) AS last_seen,
		(SELECT  count()
			FROM events e
			LEFT JOIN event_indicators ON e.id = event_indicators.event_id 
			WHERE event_indicators.indicator_id = i.id) AS events
	FROM indicators i
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Indicator{}, err
	}

	obj := Indicator{}
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) GetIndicatorByValue(cid string, value string) (Indicator, error) {
	query := `
	SELECT id, status, type, value, tlp, source, notes, case_id
	FROM indicators
	WHERE case_id = :cid AND value = :value
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid),
		sql.Named("value", value))
	if err != nil {
		return Indicator{}, err
	}

	obj := Indicator{}
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveIndicator(cid string, obj Indicator, override bool) error {
	query := `
	INSERT INTO indicators (id, status, type, value, tlp, source, notes, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :status, :type, :value, :tlp, :source, :notes, :cid)
	ON CONFLICT `
	if override {
		query += `
		DO UPDATE SET status=:status, type=:type, value=:value, tlp=:tlp, source=:source, notes=:notes
		WHERE id = :id OR (case_id = :cid AND type = :type AND value = :value)`
	} else {
		query += `
		DO NOTHING`
	}

	_, err := store.DB.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("status", obj.Status),
		sql.Named("type", obj.Type),
		sql.Named("value", obj.Value),
		sql.Named("tlp", obj.TLP),
		sql.Named("source", obj.Source),
		sql.Named("notes", obj.Notes))
	return err
}

func (store *Store) DeleteIndicator(cid string, id string) error {
	query := `
	DELETE FROM indicators
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
