package model

import (
	"gorm.io/gorm/clause"
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

	FirstSeen Time `gorm:"->"`
	LastSeen  Time `gorm:"->"`
	Events    int  `gorm:"->"`
}

func (store *Store) ListIndicators(cid string) ([]Indicator, error) {
	list := []Indicator{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("count()")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events", fsq, lsq, evq).
		Where("case_id = ?", cid).
		Order("type asc, value asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetIndicator(cid string, id string) (Indicator, error) {
	obj := Indicator{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("count()")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events", fsq, lsq, evq).
		First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) GetIndicatorByValue(cid string, value string) (Indicator, error) {
	obj := Indicator{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("count()")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events", fsq, lsq, evq).
		Where("case_id = ?", cid).
		First(&obj, "value = ?", value)
	return obj, tx.Error
}

func (store *Store) SaveIndicator(cid string, obj Indicator, override bool) error {
	return store.DB.
		Clauses(clause.OnConflict{DoNothing: !override}).
		Save(obj).
		Error
}

func (store *Store) DeleteIndicator(cid string, id string) error {
	return store.DB.Delete(&Indicator{}, "id = ?", id).Error
}
