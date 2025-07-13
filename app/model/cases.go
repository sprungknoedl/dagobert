package model

import (
	"fmt"
)

type Case struct {
	ID             string
	Name           string
	SummaryWho     string
	SummaryWhat    string
	SummaryWhen    string
	SummaryWhere   string
	SummaryWhy     string
	SummaryHow     string
	Classification string
	Severity       string
	Outcome        string
	Closed         bool

	SketchID int

	Assets     []Asset     // `gorm:"->"`
	Events     []Event     // `gorm:"->"`
	Evidences  []Evidence  // `gorm:"->"`
	Indicators []Indicator // `gorm:"->"`
	Malware    []Malware   // `gorm:"->"`
	Notes      []Note      // `gorm:"->"`
	Tasks      []Task      // `gorm:"->"`
}

func (c Case) String() string {
	if c.ID != "" {
		return fmt.Sprintf("#%s - %s", c.ID, c.Name)
	} else {
		return ""
	}
}

func (store *Store) ListCases() ([]Case, error) {
	list := []Case{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetCase(cid string) (Case, error) {
	// special case for "dumb" routes
	if cid == "" {
		return Case{}, nil
	}

	obj := Case{}
	tx := store.DB.First(&obj, "id = ?", cid)
	return obj, tx.Error
}

func (store *Store) GetCaseFull(cid string) (Case, error) {
	obj := Case{}
	tx := store.DB.
		Preload("Assets").
		Preload("Events").
		Preload("Evidences").
		Preload("Indicators").
		Preload("Malware").
		Preload("Notes").
		Preload("Tasks").
		First(&obj, "id = ?", cid)
	return obj, tx.Error
}

func (store *Store) SaveCase(obj Case) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteCase(cid string) error {
	return store.DB.Delete(Case{}, "id = ?", cid).Error
}
