package model

import (
	"database/sql"
)

var AssetTypes = FromEnv("VALUES_ASSET_TYPES", []string{"Account", "Desktop", "Server", "Other"})

type Asset struct {
	ID          string
	Type        string
	Name        string
	IP          string
	Description string
	Compromised bool
	Analysed    bool
	CaseID      string
}

func (store *Store) FindAssets(cid string, search string, sort string) ([]Asset, error) {
	query := `
	SELECT id, type, name, ip, description, compromised, analysed, case_id
	FROM assets
	WHERE case_id = :cid AND (
		instr(type, :search) > 0 OR
		instr(name, :search) > 0 OR
		instr(ip, :search) > 0 OR
		instr(description, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'analysed'     THEN analysed END ASC,
		CASE WHEN :sort = '-analysed'    THEN analysed END DESC,
		CASE WHEN :sort = 'compromised'  THEN compromised END ASC,
		CASE WHEN :sort = '-compromised' THEN compromised END DESC,
		CASE WHEN :sort = 'desc'         THEN description END ASC,
		CASE WHEN :sort = '-desc'        THEN description END DESC,
		CASE WHEN :sort = 'addr'           THEN ip END ASC,
		CASE WHEN :sort = '-addr'          THEN ip END DESC,
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
	SELECT id, type, name, ip, description, compromised, analysed, case_id
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
	REPLACE INTO assets (id, type, name, ip, description, compromised, analysed, case_id)
	VALUES (NULLIF(:id, ''), :type, :name, :ip, :description, :compromised, :analysed, :cid)`

	_, err := store.db.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("type", obj.Type),
		sql.Named("name", obj.Name),
		sql.Named("ip", obj.IP),
		sql.Named("description", obj.Description),
		sql.Named("compromised", obj.Compromised),
		sql.Named("analysed", obj.Analysed))
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
