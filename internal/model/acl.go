package model

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/sprungknoedl/dagobert/internal/fp"
)

type Policy struct {
	ID    int
	PType string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

var _ persist.Adapter = &Store{}

// LoadPolicy loads all policy rules from the storage.
func (store *Store) LoadPolicy(model model.Model) error {
	query := `
	SELECT rowid, ptype, v0, v1, v2, v3, v4, v5
	FROM policies
	`

	rows, err := store.DB.Query(query)
	if err != nil {
		return err
	}

	var list []Policy
	err = ScanAll(rows, &list)
	if err != nil {
		return err
	}

	for _, row := range list {
		arr := []string{row.PType, row.V0, row.V1, row.V2, row.V3, row.V4, row.V5}
		arr = fp.Filter(arr, func(s string) bool { return s != "" })
		err = persist.LoadPolicyArray(arr, model)
		if err != nil {
			return err
		}
	}

	return nil
}

// SavePolicy saves all policy rules to the storage.
func (store *Store) SavePolicy(model model.Model) error {
	lines := []Policy{}
	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			v := make([]string, 6)
			copy(v, rule) // resize rule (if necessary)
			lines = append(lines, Policy{PType: ptype, V0: v[0], V1: v[1], V2: v[2], V3: v[3], V4: v[4], V5: v[5]})
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			v := make([]string, 6)
			copy(v, rule) // resize rule (if necessary)
			lines = append(lines, Policy{PType: ptype, V0: v[0], V1: v[1], V2: v[2], V3: v[3], V4: v[4], V5: v[5]})
		}
	}

	tx, err := store.DB.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	_, err = tx.Exec(`DELETE FROM policies`)
	if err != nil {
		return err
	}

	for _, rule := range lines {
		query := `
		INSERT INTO policies (ptype, v0, v1, v2, v3, v4, v5)
		VALUES (:ptype, :v0, :v1, :v2, :v3, :v4, :v5)
		ON CONFLICT DO NOTHING`

		_, err := tx.Exec(query,
			sql.Named("ptype", rule.PType),
			sql.Named("v0", rule.V0),
			sql.Named("v1", rule.V1),
			sql.Named("v2", rule.V2),
			sql.Named("v3", rule.V3),
			sql.Named("v4", rule.V4),
			sql.Named("v5", rule.V5))
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}

// AddPolicy adds a policy rule to the storage.
// This is part of the Auto-Save feature.
func (store *Store) AddPolicy(sec string, ptype string, rule []string) error {
	query := `
	INSERT INTO policies (ptype, v0, v1, v2, v3, v4, v5)
	VALUES (:ptype, :v0, :v1, :v2, :v3, :v4, :v5)
	ON CONFLICT DO NOTHING`

	v := make([]string, 6)
	copy(v, rule) // resize rule (if necessary)
	_, err := store.DB.Exec(query,
		sql.Named("ptype", ptype),
		sql.Named("v0", v[0]),
		sql.Named("v1", v[1]),
		sql.Named("v2", v[2]),
		sql.Named("v3", v[3]),
		sql.Named("v4", v[4]),
		sql.Named("v5", v[5]))
	if err != nil {
		log.Printf("AddPolicy ERR = %v", err)
	}
	return err
}

// RemovePolicy removes a policy rule from the storage.
// This is part of the Auto-Save feature.
func (store *Store) RemovePolicy(sec string, ptype string, rule []string) error {
	query := `
	DELETE FROM policies
	WHERE ptype = :ptype
	  AND v0 = :v0
	  AND v1 = :v1
	  AND v2 = :v2
	  AND v3 = :v3
	  AND v4 = :v4
	  AND v5 = :v5`

	v := make([]string, 6)
	copy(v, rule) // resize rule (if necessary)
	_, err := store.DB.Exec(query,
		sql.Named("ptype", ptype),
		sql.Named("v0", v[0]),
		sql.Named("v1", v[1]),
		sql.Named("v2", v[2]),
		sql.Named("v3", v[3]),
		sql.Named("v4", v[4]),
		sql.Named("v5", v[5]))
	return err
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
// This is part of the Auto-Save feature.
func (store *Store) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	query := fmt.Sprintf(`
	DELETE FROM policies
	WHERE ptype = :ptype AND v%d = :value`, fieldIndex)

	for _, v := range fieldValues {
		_, err := store.DB.Exec(query,
			sql.Named("ptype", ptype),
			sql.Named("value", v))
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *Store) GetUserPermissions(uid string) ([]string, error) {
	query := `
	SELECT rowid, ptype, v0, v1, v2, v3, v4, v5
	FROM policies
	WHERE ptype = "p" AND v0 = :uid
	`

	rows, err := store.DB.Query(query, sql.Named("uid", uid))
	if err != nil {
		return nil, err
	}

	var list []Policy
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	cases := make([]string, 0, len(list))
	for _, row := range list {
		if strings.HasPrefix(row.V1, "/cases/") {
			kase, _, _ := strings.Cut(strings.TrimPrefix(row.V1, "/cases/"), "/")
			cases = append(cases, kase)
		}
	}

	return cases, nil
}

func (store *Store) GetCasePermissions(cid string) ([]string, error) {
	query := `
	SELECT rowid, ptype, v0, v1, v2, v3, v4, v5
	FROM policies
	WHERE ptype = "p" AND v1 = :obj
	`

	obj := fmt.Sprintf("/cases/%s/*", cid)
	rows, err := store.DB.Query(query, sql.Named("obj", obj))
	if err != nil {
		return nil, err
	}

	var list []Policy
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	users := fp.Apply(list, func(p Policy) string { return p.V0 })
	return users, nil
}
