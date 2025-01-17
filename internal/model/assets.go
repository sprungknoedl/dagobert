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

func (store *Store) ListAssets(cid string) ([]Asset, error) {
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
	WHERE case_id = :cid
	ORDER BY name ASC`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid))
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

	rows, err := store.DB.Query(query,
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

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid),
		sql.Named("name", name))
	if err != nil {
		return Asset{}, err
	}

	var obj Asset
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveAsset(cid string, obj Asset) error {
	query := `
	INSERT INTO assets (id, status, type, name, addr, notes, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :status, :type, :name, :addr, :notes, :cid)
	ON CONFLICT (id) 
		DO UPDATE SET status=:status, type=:type, name=:name, addr=:addr, notes=:notes
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("status", obj.Status),
		sql.Named("type", obj.Type),
		sql.Named("name", obj.Name),
		sql.Named("addr", obj.Addr),
		sql.Named("notes", obj.Notes))
	return err
}

func (store *Store) DeleteAsset(cid string, id string) error {
	query := `
	DELETE FROM assets
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
