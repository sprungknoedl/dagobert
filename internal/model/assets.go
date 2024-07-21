package model

import (
	"database/sql"
)

var AssetStatus = FromEnv("VALUES_ASSET_STATUS", []string{"Compromised", "Accessed", "Under investigation", "No sign of compromise", "Out of scope"})
var AssetTypes = FromEnv("VALUES_ASSET_TYPES", []string{"Account", "Desktop", "Server", "Other"})

type Asset struct {
	ID     string
	Status string
	Type   string
	Name   string
	Addr   string
	Notes  string
	CaseID string

	FirstSeen Time
	LastSeen  Time
}

func (store *Store) FindAssets(cid string, search string, sort string) ([]Asset, error) {
	query := `
	SELECT id, status, type, name, addr, notes, case_id,
		(SELECT  min(e.time)
			FROM events e
			LEFT JOIN event_assets ON e.id = event_assets.event_id 
			WHERE event_assets.asset_id = a.id) AS first_seen,
		(SELECT  max(e.time)
			FROM events e
			LEFT JOIN event_assets ON e.id = event_assets.event_id 
			WHERE event_assets.asset_id = a.id) AS last_seen
	FROM assets a
	WHERE case_id = :cid AND (
		instr(status, :search) > 0 OR
		instr(type, :search) > 0 OR
		instr(name, :search) > 0 OR
		instr(addr, :search) > 0 OR
		instr(notes, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'notes'        THEN notes END ASC,
		CASE WHEN :sort = '-notes'       THEN notes END DESC,
		CASE WHEN :sort = 'addr'         THEN addr END ASC,
		CASE WHEN :sort = '-addr'        THEN addr END DESC,
		CASE WHEN :sort = 'status'         THEN status END ASC,
		CASE WHEN :sort = '-status'        THEN status END DESC,
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

	var list []Asset
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetAsset(cid string, id string) (Asset, error) {
	query := `
	SELECT id, status, type, name, addr, notes, case_id,
		(SELECT  min(e.time)
			FROM events e
			LEFT JOIN event_assets ON e.id = event_assets.event_id 
			WHERE event_assets.asset_id = a.id) AS first_seen,
		(SELECT  max(e.time)
			FROM events e
			LEFT JOIN event_assets ON e.id = event_assets.event_id 
			WHERE event_assets.asset_id = a.id) AS last_seen
	FROM assets a
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Asset{}, err
	}

	var obj Asset
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) GetAssetByName(cid string, name string) (Asset, error) {
	query := `
	SELECT id, status, type, name, addr, notes, case_id
	FROM assets
	WHERE case_id = :cid AND name = :name
	LIMIT 1`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("name", name))
	if err != nil {
		return Asset{}, err
	}

	var obj Asset
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveAsset(cid string, obj Asset) (Asset, error) {
	query := `
	INSERT INTO assets (id, status, type, name, addr, notes, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :status, :type, :name, :addr, :notes, :cid)
	ON CONFLICT (id) 
		DO UPDATE SET status=:status, type=:type, name=:name, addr=:addr, notes=:notes
		WHERE id = :id
	RETURNING id, status, type, name, addr, notes, case_id`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("status", obj.Status),
		sql.Named("type", obj.Type),
		sql.Named("name", obj.Name),
		sql.Named("addr", obj.Addr),
		sql.Named("notes", obj.Notes))
	if err != nil {
		return Asset{}, err
	}

	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) DeleteAsset(cid string, id string) error {
	query := `
	DELETE FROM assets
	WHERE id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
