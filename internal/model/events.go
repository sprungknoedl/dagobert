package model

import (
	"database/sql"
	"encoding/json"
	"slices"
	"time"
)

var EventTypes = FromEnv("VALUES_EVENT_TYPES", []string{
	"Reconnaissance",
	"Resource Development",
	"Initial Access",
	"Execution",
	"Persistence",
	"Privilege Escalation",
	"Defense Evasion",
	"Credential Access",
	"Discovery",
	"Lateral Movement",
	"Collection",
	"C2",
	"Exfiltration",
	"Impact",
	"Legitimate",
	"Remediation",
	"Other",
})

type Event struct {
	ID            string
	Time          time.Time
	Type          string
	Event         string
	Raw           string
	CaseID        string
	RawAssets     []byte
	RawIndicators []byte

	Assets     []Asset
	Indicators []Indicator
}

func (e Event) HasAsset(aid string) bool {
	for _, a := range e.Assets {
		if a.ID == aid {
			return true
		}
	}
	return false
}

func (e Event) HasIndicator(iid string) bool {
	for _, i := range e.Indicators {
		if i.ID == iid {
			return true
		}
	}
	return false
}

func (store *Store) FindEvents(cid string, search string, sort string) ([]Event, error) {
	query := `
	SELECT 
		e.id, e.time, e.type, e.event, e.raw, e.case_id,
		(SELECT json_group_array(json_object('ID', a.id, 'Type', a.type, 'Name', a.name))
			FROM assets a
			LEFT JOIN event_assets ON a.id = event_assets.asset_id 
			WHERE event_assets.event_id = e.id) AS assets,
		(SELECT json_group_array(json_object('ID', i.id, 'Type', i.type, 'Value', i.value))
			FROM indicators i
			LEFT JOIN event_indicators ON i.id = event_indicators.indicator_id 
			WHERE event_indicators.event_id = e.id) AS indicators
	FROM events e
	WHERE
		e.case_id = :cid AND (
		instr(e.type, :search) > 0 OR
		instr(e.event, :search) > 0 OR
		instr(e.raw, :search) > 0)
		-- instr(assets->>'$.Name', :search) > 0 OR
		-- instr(indicators->>'$.Value', :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'time'        THEN e.time END ASC,
		CASE WHEN :sort = '-time'       THEN e.time END DESC,
		CASE WHEN :sort = 'type'        THEN e.type END ASC,
		CASE WHEN :sort = '-type'       THEN e.type END DESC,
		CASE WHEN :sort = 'event'       THEN e.event END ASC,
		CASE WHEN :sort = '-event'      THEN e.event END DESC,
		-- CASE WHEN :sort = 'assets'      THEN assets->>'$.Name' END ASC,
		-- CASE WHEN :sort = '-assets'     THEN assets->>'$.Name' END ASC,
		-- CASE WHEN :sort = 'indicators'  THEN indicators->>'$.Value' END DESC,
		-- CASE WHEN :sort = '-indicators' THEN indicators->>'$.Value' END DESC,
		e.time ASC`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("search", search),
		sql.Named("sort", sort))
	if err != nil {
		return nil, err
	}

	var list []Event
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	// unmarshal json encoded relations
	for i, elem := range list {
		err = json.Unmarshal(elem.RawAssets, &elem.Assets)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(elem.RawIndicators, &elem.Indicators)
		if err != nil {
			return nil, err
		}

		elem.Assets = slices.DeleteFunc(elem.Assets, func(a Asset) bool { return a.ID == "" })
		elem.Indicators = slices.DeleteFunc(elem.Indicators, func(i Indicator) bool { return i.ID == "" })
		list[i] = elem
	}

	return list, err
}

func (store *Store) GetEvent(cid string, id string) (Event, error) {
	query := `
	SELECT 
		e.id, e.time, e.type, e.event, e.raw, e.case_id,
		(SELECT json_group_array(json_object('ID', a.id, 'Type', a.type, 'Name', a.name))
			FROM assets a
			LEFT JOIN event_assets ON a.id = event_assets.asset_id 
			WHERE event_assets.event_id = e.id) AS assets,
		(SELECT json_group_array(json_object('ID', i.id, 'Type', i.type, 'Value', i.value))
			FROM indicators i
			LEFT JOIN event_indicators ON i.id = event_indicators.indicator_id 
			WHERE event_indicators.event_id = e.id) AS indicators
	FROM events e
	WHERE e.id = :id
	LIMIT 1`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Event{}, err
	}

	var obj Event
	err = ScanOne(rows, &obj)
	if err != nil {
		return Event{}, err
	}

	// unmarshal json encoded relations
	err = json.Unmarshal(obj.RawAssets, &obj.Assets)
	if err != nil {
		return Event{}, err
	}
	err = json.Unmarshal(obj.RawIndicators, &obj.Indicators)
	if err != nil {
		return Event{}, err
	}

	return obj, nil
}

func (store *Store) SaveEvent(cid string, obj Event) error {
	query := `
	REPLACE INTO events (id, time, type, event, raw, case_id)
	VALUES (NULLIF(:id, ''), :time, :type, :event, :raw, :cid)
	RETURNING id, time, type, event, raw, case_id`

	// assets
	query2 := `
	DELETE FROM event_assets
	WHERE event_id = :eid`
	query3 := `
	REPLACE INTO event_assets (event_id, asset_id)
	VALUES (:eid, :aid)`

	// indicators
	query4 := `
	DELETE FROM event_indicators
	WHERE event_id = :eid`
	query5 := `
	REPLACE INTO event_indicators (event_id, indicator_id)
	VALUES (:eid, :iid)`

	tx, err := store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("time", obj.Time),
		sql.Named("type", obj.Type),
		sql.Named("event", obj.Event),
		sql.Named("raw", obj.Raw))
	if err != nil {
		return err
	}

	if err = ScanOne(rows, &obj); err != nil {
		return err
	}

	// assets
	_, err = tx.Exec(query2,
		sql.Named("eid", obj.ID))
	if err != nil {
		return err
	}
	for _, a := range obj.Assets {
		_, err := tx.Exec(query3,
			sql.Named("eid", obj.ID),
			sql.Named("aid", a.ID))
		if err != nil {
			return err
		}
	}

	// indicators
	_, err = tx.Exec(query4,
		sql.Named("eid", obj.ID))
	if err != nil {
		return err
	}
	for _, i := range obj.Indicators {
		_, err := tx.Exec(query5,
			sql.Named("eid", obj.ID),
			sql.Named("iid", i.ID))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (store *Store) DeleteEvent(cid string, id string) error {
	query := `
	DELETE FROM events
	WHERE events.id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
