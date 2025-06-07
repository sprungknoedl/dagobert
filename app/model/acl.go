package model

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"gorm.io/gorm"
)

type Policy struct {
	Ptype string `gorm:"primaryKey"`
	V0    string `gorm:"primaryKey"`
	V1    string `gorm:"primaryKey"`
	V2    string `gorm:"primaryKey"`
	V3    string `gorm:"primaryKey"`
	V4    string `gorm:"primaryKey"`
	V5    string `gorm:"primaryKey"`
}

func (Policy) TableName() string { return "policies" }

var _ persist.Adapter = &Store{}

// LoadPolicy loads all policy rules from the storage.
func (store *Store) LoadPolicy(model model.Model) error {
	list := []Policy{}
	err := store.DB.Find(&list).Error
	if err != nil {
		return err
	}

	for _, row := range list {
		arr := []string{row.Ptype, row.V0, row.V1, row.V2, row.V3, row.V4, row.V5}
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
			lines = append(lines, Policy{Ptype: ptype, V0: v[0], V1: v[1], V2: v[2], V3: v[3], V4: v[4], V5: v[5]})
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			v := make([]string, 6)
			copy(v, rule) // resize rule (if necessary)
			lines = append(lines, Policy{Ptype: ptype, V0: v[0], V1: v[1], V2: v[2], V3: v[3], V4: v[4], V5: v[5]})
		}
	}

	return store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(Policy{}).Error; err != nil {
			return err
		}

		return tx.Create(lines).Error
	})
}

// AddPolicy adds a policy rule to the storage.
// This is part of the Auto-Save feature.
func (store *Store) AddPolicy(sec string, ptype string, rule []string) error {
	v := make([]string, 6)
	copy(v, rule) // resize rule (if necessary)
	obj := Policy{Ptype: ptype, V0: v[0], V1: v[1], V2: v[2], V3: v[3], V4: v[4], V5: v[5]}
	return store.DB.Save(&obj).Error
}

// RemovePolicy removes a policy rule from the storage.
// This is part of the Auto-Save feature.
func (store *Store) RemovePolicy(sec string, ptype string, rule []string) error {
	v := make([]string, 6)
	copy(v, rule) // resize rule (if necessary)
	obj := Policy{Ptype: ptype, V0: v[0], V1: v[1], V2: v[2], V3: v[3], V4: v[4], V5: v[5]}
	return store.DB.Delete(&obj).Error
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
// This is part of the Auto-Save feature.
func (store *Store) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	filter := fmt.Sprintf("ptype = ? AND v%d = ?", fieldIndex)
	for _, v := range fieldValues {
		err := store.DB.Delete(Policy{}, filter, ptype, v).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *Store) GetUserPermissions(uid string) ([]string, error) {
	list := []Policy{}
	err := store.DB.Find(&list, "ptype = ? and v0 = ?", "p", uid).Error
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
	list := []Policy{}
	err := store.DB.Find(&list, "ptype = ? and v1 = ?", "p", fmt.Sprintf("/cases/%s/*", cid)).Error
	if err != nil {
		return nil, err
	}

	users := fp.Apply(list, func(p Policy) string { return p.V0 })
	return users, nil
}
