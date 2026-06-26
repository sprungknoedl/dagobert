package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Indicator struct {
	ID      string
	Flagged bool
	Status  string
	Type    string
	Value   string
	TLP     string
	Source  string
	Notes   string
	CaseID  string
	Custom  Custom `form:"-"`

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

// SetIndicatorCustom merges the given label→value fields into the indicator's
// custom column in a single statement. Enrichment modules call this to write
// their attributes back; running one json_set per save keeps parallel modules
// from clobbering each other's keys (the read-then-write of a load/Save cycle
// would race even though SQLite serializes writes).
//
// Empty values are skipped (custom-attributes' "empty value = no key" rule), so
// a "not found" result that omits a key never stores "". The JSON paths are
// bound as parameters — labels contain spaces ("MISP Enrichment"), so they must
// be quoted members, never string-concatenated into the SQL.
func (store *Store) SetIndicatorCustom(cid, id string, fields map[string]string) error {
	// seed handles the NOT NULL DEFAULT '' column: json_set('', …) errors
	// because '' is not valid JSON, so upgrade an empty column to '{}' first.
	sql := "json_set(CASE WHEN custom IS NULL OR custom = '' THEN '{}' ELSE custom END"
	args := []any{}
	for label, value := range fields {
		if value == "" {
			continue
		}
		sql += ", ?, ?"
		args = append(args, `$."`+label+`"`, value)
	}
	sql += ")"

	if len(args) == 0 {
		return nil
	}

	return store.DB.Model(&Indicator{}).
		Where("id = ? AND case_id = ?", id, cid).
		Update("custom", gorm.Expr(sql, args...)).
		Error
}
