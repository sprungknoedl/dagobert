package model

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Event struct {
	ID            string `gorm:"primaryKey"`
	Time          Time
	Type          string
	Event         string
	Raw           string
	Flagged       bool
	CaseID        string
	RawAssets     []byte `gorm:"-"`
	RawIndicators []byte `gorm:"-"`

	Assets     []Asset     `gorm:"many2many:event_assets;"`
	Indicators []Indicator `gorm:"many2many:event_indicators;"`
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

func (store *Store) ListEvents(cid string) ([]Event, error) {
	list := []Event{}
	tx := store.DB.
		Preload("Assets").
		Preload("Indicators").
		Where("case_id = ?", cid).
		Order("time asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetEvent(cid string, id string) (Event, error) {
	obj := Event{}
	tx := store.DB.
		Preload("Assets").
		Preload("Indicators").
		First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveEvent(cid string, obj Event, override bool) error {
	return store.DB.Transaction(func(tx *gorm.DB) error {
		return errors.Join(
			tx.
				Clauses(clause.OnConflict{DoNothing: !override, UpdateAll: override}).
				Omit("Assets").
				Omit("Indicators").
				Create(&obj).
				Error,
			tx.Model(&obj).Association("Assets").Replace(obj.Assets),
			tx.Model(&obj).Association("Indicators").Replace(obj.Indicators),
		)
	})
}

func (store *Store) DeleteEvent(cid string, id string) error {
	return store.DB.Delete(&Event{}, "id = ?", id).Error
}
