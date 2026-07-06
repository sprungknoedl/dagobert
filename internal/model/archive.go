package model

import (
	"fmt"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

// ArchiveFormat and ArchiveFormatVersion identify the case-archive layout. The
// format version is bumped only when the archive structure itself changes;
// import rejects any other value.
const (
	ArchiveFormat        = "dagobert-case"
	ArchiveFormatVersion = 1
)

// Manifest is the small, self-describing header read first (and on its own)
// from a case archive so a version/format check can reject an incompatible
// archive before the full graph is decoded.
type Manifest struct {
	Format           string         `json:"format"`
	FormatVersion    int            `json:"formatVersion"`
	SchemaVersion    uint           `json:"schemaVersion"`
	ExportedAt       Time           `json:"exportedAt"`
	ExportedBy       string         `json:"exportedBy"`
	SourceInstance   string         `json:"sourceInstance"`
	IncludesBinaries bool           `json:"includesBinaries"`
	OriginalCaseID   string         `json:"originalCaseID"`
	Counts           map[string]int `json:"counts"`
	Warnings         []string       `json:"warnings,omitempty"`
}

// CaseArchive is the serialized object graph of a single case (case.json). The
// many2many event links are flattened onto each event as id lists rather than
// as separate join arrays.
type CaseArchive struct {
	Case       Case           `json:"case"`
	Assets     []Asset        `json:"assets"`
	Indicators []Indicator    `json:"indicators"`
	Events     []ArchiveEvent `json:"events"`
	Malware    []Malware      `json:"malware"`
	Notes      []Note         `json:"notes"`
	Tasks      []Task         `json:"tasks"`
	Evidences  []Evidence     `json:"evidences"`
	Comments   []Comment      `json:"comments"`
}

// ArchiveEvent is the export projection of an Event: the event itself plus the
// foreign-key id lists of its associated assets and indicators. Keeping these
// here (rather than on model.Event) keeps the GORM model clean.
type ArchiveEvent struct {
	Event
	AssetIDs     []string
	IndicatorIDs []string
}

// ExportCaseArchive builds the full, binary-free object graph of a case for
// serialization into case.json. Computed and relation fields that would
// duplicate or pollute the archive are cleared; the event many2many links are
// projected into id lists.
func (store *Store) ExportCaseArchive(cid string) (CaseArchive, error) {
	obj, err := store.GetCaseFull(cid)
	if err != nil {
		return CaseArchive{}, err
	}

	// ListEvents preloads the Assets/Indicators associations (GetCaseFull does
	// not nest-preload them), which we need to project the join id lists.
	events, err := store.ListEvents(cid)
	if err != nil {
		return CaseArchive{}, err
	}

	comments := []Comment{}
	if err := store.DB.Where("case_id = ?", cid).Order("time asc").Find(&comments).Error; err != nil {
		return CaseArchive{}, err
	}

	// the case row carries the scalar fields only; the child collections live in
	// their own top-level arrays
	kase := obj
	kase.Assets = nil
	kase.Events = nil
	kase.Evidences = nil
	kase.Indicators = nil
	kase.Malware = nil
	kase.Notes = nil
	kase.Tasks = nil

	arch := CaseArchive{
		Case:       kase,
		Assets:     obj.Assets,
		Indicators: obj.Indicators,
		Malware:    obj.Malware,
		Notes:      obj.Notes,
		Tasks:      obj.Tasks,
		Evidences:  obj.Evidences,
		Comments:   comments,
		Events: fp.Apply(events, func(e Event) ArchiveEvent {
			ae := ArchiveEvent{
				AssetIDs:     fp.Apply(e.Assets, func(a Asset) string { return a.ID }),
				IndicatorIDs: fp.Apply(e.Indicators, func(i Indicator) string { return i.ID }),
			}
			e.Assets = nil
			e.Indicators = nil
			ae.Event = e
			return ae
		}),
	}
	return arch, nil
}

// ImportCaseArchive recreates a case and every child record from an archive,
// preserving all ids verbatim. It first fails the whole import if any id in the
// archive already exists on this instance, then inserts in dependency order. The
// collision check and inserts run in one transaction so any failure leaves no
// partial case behind.
func (store *Store) ImportCaseArchive(arch CaseArchive) error {
	return store.Transaction(func(tx *Store) error {
		if err := tx.assertNoCollisions(arch); err != nil {
			return err
		}

		// order matters for the foreign keys and join links:
		// case -> evidences/assets/indicators -> events (+joins) -> malware -> notes -> tasks
		if err := tx.SaveCase(arch.Case); err != nil {
			return err
		}
		for _, e := range arch.Evidences {
			if err := tx.SaveEvidence(arch.Case.ID, e); err != nil {
				return err
			}
		}
		for _, a := range arch.Assets {
			if err := tx.SaveAsset(arch.Case.ID, a); err != nil {
				return err
			}
		}
		for _, i := range arch.Indicators {
			if err := tx.SaveIndicator(arch.Case.ID, i, false); err != nil {
				return err
			}
		}
		for _, ae := range arch.Events {
			ev := ae.Event
			// stub records carrying just the archive ids; the assets/indicators
			// themselves are already inserted above, so SaveEvent only writes the
			// join rows (GORM does not full-save associations by default)
			ev.Assets = fp.Apply(ae.AssetIDs, func(id string) Asset { return Asset{ID: id} })
			ev.Indicators = fp.Apply(ae.IndicatorIDs, func(id string) Indicator { return Indicator{ID: id} })
			if err := tx.SaveEvent(arch.Case.ID, ev, false); err != nil {
				return err
			}
		}
		for _, m := range arch.Malware {
			// SaveMalware derives AssetID from Asset.ID, so carry the id through
			// the stub relation to preserve the malware->asset reference
			if m.AssetID != nil {
				m.Asset = Asset{ID: *m.AssetID}
			}
			if err := tx.SaveMalware(arch.Case.ID, m); err != nil {
				return err
			}
		}
		for _, n := range arch.Notes {
			if err := tx.SaveNote(arch.Case.ID, n); err != nil {
				return err
			}
		}
		for _, t := range arch.Tasks {
			if err := tx.SaveTask(arch.Case.ID, t); err != nil {
				return err
			}
		}
		for _, c := range arch.Comments {
			if err := tx.SaveComment(arch.Case.ID, c); err != nil {
				return err
			}
		}
		return nil
	})
}

// ForkCase duplicates a case into dst by round-tripping the archive
// export/import: every child row gets a fresh id (relations remapped via one
// old→new map), and the archive's case row is replaced with the form-submitted
// dst. The import runs in its usual transaction, so a failed fork leaves
// nothing behind. Files on disk and ACL rules are the caller's job.
func (store *Store) ForkCase(srcID string, dst Case) (Case, error) {
	arch, err := store.ExportCaseArchive(srcID)
	if err != nil {
		return Case{}, err
	}

	ids := map[string]string{arch.Case.ID: dst.ID}
	remap := func(old string) string {
		if ids[old] == "" {
			ids[old] = fp.Random(10)
		}
		return ids[old]
	}

	arch.Case = dst
	for i := range arch.Assets {
		arch.Assets[i].ID = remap(arch.Assets[i].ID)
		arch.Assets[i].CaseID = dst.ID
	}
	for i := range arch.Indicators {
		arch.Indicators[i].ID = remap(arch.Indicators[i].ID)
		arch.Indicators[i].CaseID = dst.ID
	}
	for i := range arch.Events {
		arch.Events[i].ID = remap(arch.Events[i].ID)
		arch.Events[i].CaseID = dst.ID
		arch.Events[i].AssetIDs = fp.Apply(arch.Events[i].AssetIDs, remap)
		arch.Events[i].IndicatorIDs = fp.Apply(arch.Events[i].IndicatorIDs, remap)
	}
	for i := range arch.Malware {
		arch.Malware[i].ID = remap(arch.Malware[i].ID)
		arch.Malware[i].CaseID = dst.ID
		if arch.Malware[i].AssetID != nil {
			id := remap(*arch.Malware[i].AssetID)
			arch.Malware[i].AssetID = &id
		}
	}
	for i := range arch.Notes {
		arch.Notes[i].ID = remap(arch.Notes[i].ID)
		arch.Notes[i].CaseID = dst.ID
	}
	for i := range arch.Tasks {
		arch.Tasks[i].ID = remap(arch.Tasks[i].ID)
		arch.Tasks[i].CaseID = dst.ID
	}
	for i := range arch.Evidences {
		arch.Evidences[i].ID = remap(arch.Evidences[i].ID)
		arch.Evidences[i].CaseID = dst.ID
	}
	for i := range arch.Comments {
		arch.Comments[i].ID = remap(arch.Comments[i].ID)
		arch.Comments[i].CaseID = dst.ID
		arch.Comments[i].ObjectID = remap(arch.Comments[i].ObjectID)
	}

	return dst, store.ImportCaseArchive(arch)
}

// assertNoCollisions fails the import if any id in the archive already exists on
// this instance, with a message naming the table and the colliding id(s).
func (store *Store) assertNoCollisions(arch CaseArchive) error {
	if found, err := store.existingIDs("cases", []string{arch.Case.ID}); err != nil {
		return err
	} else if len(found) > 0 {
		return fmt.Errorf("case %q already exists; delete it first or import a different case", arch.Case.ID)
	}

	checks := []struct {
		table string
		ids   []string
	}{
		{"assets", fp.Apply(arch.Assets, func(a Asset) string { return a.ID })},
		{"indicators", fp.Apply(arch.Indicators, func(i Indicator) string { return i.ID })},
		{"events", fp.Apply(arch.Events, func(e ArchiveEvent) string { return e.ID })},
		{"malware", fp.Apply(arch.Malware, func(m Malware) string { return m.ID })},
		{"notes", fp.Apply(arch.Notes, func(n Note) string { return n.ID })},
		{"tasks", fp.Apply(arch.Tasks, func(t Task) string { return t.ID })},
		{"evidences", fp.Apply(arch.Evidences, func(e Evidence) string { return e.ID })},
		{"comments", fp.Apply(arch.Comments, func(c Comment) string { return c.ID })},
	}
	for _, c := range checks {
		found, err := store.existingIDs(c.table, c.ids)
		if err != nil {
			return err
		}
		if len(found) > 0 {
			return fmt.Errorf("import aborted: %s id(s) already exist on this instance: %v", c.table, found)
		}
	}
	return nil
}

// existingIDs returns the subset of ids that already exist in the given table.
func (store *Store) existingIDs(table string, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var found []string
	err := store.DB.Table(table).Where("id IN ?", ids).Pluck("id", &found).Error
	return found, err
}
