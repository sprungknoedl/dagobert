package model

import (
	"fmt"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

type Case struct {
	ID             string
	Name           string
	Summary        string
	Classification string
	Severity       string
	Outcome        string
	Closed         bool
	IsTemplate     bool
	OpenedAt       Date
	ClosedAt       Date
	Custom         Custom `form:"-"`

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
		Where("is_template = ?", false).
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) ListTemplates() ([]Case, error) {
	list := []Case{}
	tx := store.DB.
		Where("is_template = ?", true).
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

// CloneCaseContents creates dst (which the caller has populated with a fresh ID,
// the IsTemplate flag, a name, and the three case-level defaults) and copies the
// source case's tasks and notes into it. Findings are never copied. Each cloned
// row gets a fresh ID and the destination CaseID; tasks are reset to not-done
// with a blank due date. It returns the saved destination case.
func (store *Store) CloneCaseContents(srcID string, dst Case) (Case, error) {
	// run every insert in one transaction so a failure half-way leaves no
	// partially-populated case behind
	err := store.Transaction(func(tx *Store) error {
		if err := tx.SaveCase(dst); err != nil {
			return err
		}

		tasks, err := tx.ListTasks(srcID)
		if err != nil {
			return err
		}
		for _, t := range tasks {
			t.ID = fp.Random(10)
			t.CaseID = dst.ID
			t.Done = false
			t.DateDue = Time{}
			if err := tx.SaveTask(dst.ID, t); err != nil {
				return err
			}
		}

		notes, err := tx.ListNotes(srcID)
		if err != nil {
			return err
		}
		for _, n := range notes {
			n.ID = fp.Random(10)
			n.CaseID = dst.ID
			if err := tx.SaveNote(dst.ID, n); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return Case{}, err
	}

	return dst, nil
}

func (store *Store) GetCase(cid string) (Case, error) {
	// special case for "dumb" routes and "add" forms, where the {cid} path
	// value is empty or the "new" sentinel rather than a real case id
	if cid == "" || cid == "new" {
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
	return store.Transaction(func(tx *Store) error {
		// comments carry no foreign key, so the case FK cascade does not reach them
		if err := tx.DB.Delete(&Comment{}, "case_id = ?", cid).Error; err != nil {
			return err
		}
		return tx.DB.Delete(Case{}, "id = ?", cid).Error
	})
}
