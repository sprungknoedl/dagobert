package model

import (
	"strings"

	"github.com/sprungknoedl/dagobert/pkg/fp"
	"gorm.io/gorm/clause"
)

// EnrichmentVerdicts is the strict ordered set of recognised verdict tokens,
// highest severity first. Shared between the view helper and enrichment modules
// so the vocabulary can't drift.
var EnrichmentVerdicts = []string{"malicious", "suspicious", "clean", "unknown"}

var verdictRank = map[string]int{
	"malicious":  3,
	"suspicious": 2,
	"clean":      1,
	"unknown":    0,
}

// VerdictSeverity reports the severity rank (malicious=3 … unknown=0) and
// whether v is a recognised verdict. Matching is case-insensitive.
func VerdictSeverity(v string) (rank int, ok bool) {
	rank, ok = verdictRank[strings.ToLower(v)]
	return
}

// Enrichment is machine-generated threat intelligence about an object, keyed by
// (object, module). It is a re-derivable cache: workers repopulate it on every
// run and analysts never edit it. One row per (ObjectType, ObjectID, Module).
type Enrichment struct {
	ID         string `gorm:"primaryKey"`
	CaseID     string
	ObjectType string // "Indicator" | "Evidence" | "Malware" | ...
	ObjectID   string
	Module     string // "VirusTotal"
	Verdict    string // malicious|suspicious|clean|unknown|"" — optional
	Summary    string // human-readable, always present
	Link       string // optional deep link
	FetchedAt  Time
}

// SetEnrichment upserts an enrichment row on the (object_type, object_id,
// module) unique index — re-running a module replaces its previous result. The
// ID is generated when empty.
func (store *Store) SetEnrichment(e Enrichment) error {
	if e.ID == "" {
		e.ID = fp.Random(10)
	}
	return store.DB.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "object_type"}, {Name: "object_id"}, {Name: "module"}},
			UpdateAll: true,
		}).
		Create(&e).
		Error
}

// ListEnrichments returns an object's enrichment rows ordered by module.
func (store *Store) ListEnrichments(objectType, objectID string) ([]Enrichment, error) {
	list := []Enrichment{}
	tx := store.DB.
		Where("object_type = ? AND object_id = ?", objectType, objectID).
		Order("module asc").
		Find(&list)
	return list, tx.Error
}

// ListEnrichmentsForCase loads every enrichment row of one object type in a case
// in a single query and groups them by ObjectID for the list view.
func (store *Store) ListEnrichmentsForCase(caseID, objectType string) (map[string][]Enrichment, error) {
	list := []Enrichment{}
	tx := store.DB.
		Where("case_id = ? AND object_type = ?", caseID, objectType).
		Order("module asc").
		Find(&list)
	if tx.Error != nil {
		return nil, tx.Error
	}

	out := map[string][]Enrichment{}
	for _, e := range list {
		out[e.ObjectID] = append(out[e.ObjectID], e)
	}
	return out, nil
}

// DeleteEnrichments removes all enrichment rows for an object. Called when the
// object itself is deleted.
func (store *Store) DeleteEnrichments(objectType, objectID string) error {
	return store.DB.Delete(&Enrichment{}, "object_type = ? AND object_id = ?", objectType, objectID).Error
}
