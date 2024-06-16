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
}

func (store *Store) FindAssets(cid string, search string, sort string) ([]Asset, error) {
	query := `
	SELECT id, status, type, name, addr, notes, case_id
	FROM assets
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
	SELECT id, status, type, name, addr, notes, case_id
	FROM assets
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

func (store *Store) SaveAsset(cid string, obj Asset) error {
	query := `
	REPLACE INTO assets (id, status, type, name, addr, notes, case_id)
	VALUES (NULLIF(:id, ''), :status, :type, :name, :addr, :notes, :cid)`

	_, err := store.db.Exec(query,
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

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
