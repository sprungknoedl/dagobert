package model

import (
	"errors"
	"time"

	"github.com/sprungknoedl/dagobert/pkg/fp"
	"gorm.io/gorm"
)

type Evidence struct {
	ID       string
	Type     string
	Name     string
	Hash     string
	Size     int64
	Source   string
	Notes    string
	Password string
	CaseID   string
	StartsAt Time
	EndsAt   Time
	Custom   Custom `form:"-"`
}

func (store *Store) ListEvidences(cid string) ([]Evidence, error) {
	list := []Evidence{}
	tx := store.DB.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetEvidence(cid string, id string) (Evidence, error) {
	obj := Evidence{}
	tx := store.DB.First(&obj, "id = ? AND case_id = ?", id, cid)
	return obj, tx.Error
}

func (store *Store) SaveEvidence(cid string, obj Evidence) error {
	obj.CaseID = cid
	if err := store.assertCaseOwnership(&Evidence{}, obj.ID, cid); err != nil {
		return err
	}
	return store.DB.Save(&obj).Error
}

// DeleteEvidence removes the evidence row and its comments/enrichments, and
// writes the "deleted" log entry in the same transaction — actor is the
// display string of the user performing the delete.
func (store *Store) DeleteEvidence(cid string, id string, actor string) error {
	return store.Transaction(func(tx *Store) error {
		obj, err := tx.GetEvidence(cid, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		res := tx.DB.Delete(Evidence{}, "id = ? AND case_id = ?", id, cid)
		if res.Error != nil || res.RowsAffected == 0 {
			return res.Error
		}
		if err := tx.deleteObjectComments(cid, "evidences", id); err != nil {
			return err
		}
		if err := tx.DeleteEnrichments("Evidence", id); err != nil {
			return err
		}

		return tx.SaveEvidenceLog(cid, EvidenceLog{
			EvidenceID: id,
			Name:       obj.Name,
			User:       actor,
			Event:      EvidenceLogDeleted,
		})
	})
}

// EvidenceLog event constants — fixed, not a customizable value list.
const (
	EvidenceLogUploaded   = "uploaded"
	EvidenceLogDownloaded = "downloaded"
	EvidenceLogEdited     = "edited"
	EvidenceLogModuleRun  = "module run"
	EvidenceLogDeleted    = "deleted"
)

// EvidenceLog is an append-only, self-contained record of an access-relevant
// action on an Evidence row. Rows carry no FKs and denormalize the evidence
// name and acting user, so they stay meaningful after either is deleted.
type EvidenceLog struct {
	ID         string
	CaseID     string
	EvidenceID string
	Name       string
	User       string
	Event      string
	Details    string
	Time       Time
}

// ListEvidenceLogs returns a case's log rows, newest first.
func (store *Store) ListEvidenceLogs(cid string) ([]EvidenceLog, error) {
	list := []EvidenceLog{}
	tx := store.DB.
		Where("case_id = ?", cid).
		Order("time desc").
		Find(&list)
	return list, tx.Error
}

// SaveEvidenceLog inserts a new, immutable log row. ID and Time are filled in
// when left zero, so call sites only need to set the descriptive fields.
func (store *Store) SaveEvidenceLog(cid string, obj EvidenceLog) error {
	obj.CaseID = cid
	if obj.ID == "" {
		obj.ID = fp.Random(10)
	}
	if obj.Time.IsZero() {
		obj.Time = Time(time.Now())
	}
	return store.DB.Create(&obj).Error
}

// PurgeEvidenceLogs permanently removes all log rows for one (usually
// deleted) evidence.
func (store *Store) PurgeEvidenceLogs(cid, evidenceID string) error {
	return store.DB.Delete(&EvidenceLog{}, "case_id = ? AND evidence_id = ?", cid, evidenceID).Error
}
