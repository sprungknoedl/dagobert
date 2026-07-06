package model

import (
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

	FirstSeen  Time `gorm:"->"`
	LastSeen   Time `gorm:"->"`
	Events     int  `gorm:"->"`
	OtherCases int  `gorm:"->"`
}

// IndicatorCaseRef identifies another case that shares an indicator value/type
// with the current one. Used by the cross-case "related cases" panel.
type IndicatorCaseRef struct {
	ID       string
	Name     string
	Severity string
	Closed   bool
}

func (store *Store) ListIndicators(cid string) ([]Indicator, error) {
	list := []Indicator{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_indicators ON events.id = event_indicators.event_id").Where("event_indicators.indicator_id = indicators.id").Select("count()")
	ocq := store.DB.Table("indicators i2").
		Joins("JOIN cases c2 ON c2.id = i2.case_id").
		Where("i2.value = indicators.value COLLATE NOCASE AND i2.type = indicators.type AND i2.case_id <> indicators.case_id AND c2.is_template = 0").
		Select("count(DISTINCT i2.case_id)")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events, (?) as other_cases", fsq, lsq, evq, ocq).
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
	ocq := store.DB.Table("indicators i2").
		Joins("JOIN cases c2 ON c2.id = i2.case_id").
		Where("i2.value = indicators.value COLLATE NOCASE AND i2.type = indicators.type AND i2.case_id <> indicators.case_id AND c2.is_template = 0").
		Select("count(DISTINCT i2.case_id)")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events, (?) as other_cases", fsq, lsq, evq, ocq).
		First(&obj, "id = ? AND case_id = ?", id, cid)
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

// ListIndicatorOverlap returns the distinct non-template cases (other than
// excludeCid) that contain an indicator with the same type and value. The match
// on value is case-insensitive. The full (unfiltered) set is returned; ACL
// filtering happens in the handler.
func (store *Store) ListIndicatorOverlap(excludeCid, typ, value string) ([]IndicatorCaseRef, error) {
	list := []IndicatorCaseRef{}
	tx := store.DB.Table("cases").
		Joins("JOIN indicators ON indicators.case_id = cases.id").
		Where("indicators.value = ? COLLATE NOCASE AND indicators.type = ? AND cases.id <> ? AND cases.is_template = 0", value, typ, excludeCid).
		Select("DISTINCT cases.id as id, cases.name as name, cases.severity as severity, cases.closed as closed").
		Order("cases.closed asc, cases.severity asc, cases.name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) SaveIndicator(cid string, obj Indicator, override bool) error {
	obj.CaseID = cid
	if err := store.assertCaseOwnership(&Indicator{}, obj.ID, cid); err != nil {
		return err
	}
	return store.DB.
		Clauses(clause.OnConflict{DoNothing: !override}).
		Save(obj).
		Error
}

func (store *Store) DeleteIndicator(cid string, id string) error {
	return store.Transaction(func(tx *Store) error {
		res := tx.DB.Delete(&Indicator{}, "id = ? AND case_id = ?", id, cid)
		if res.Error != nil || res.RowsAffected == 0 {
			return res.Error
		}
		if err := tx.deleteObjectComments(cid, "indicators", id); err != nil {
			return err
		}
		return tx.DeleteEnrichments("Indicator", id)
	})
}
