package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestCrossCaseIsolation locks in the fix for the systemic cross-case IDOR:
// Get/Delete are scoped by case_id, and Save refuses to upsert over a record
// owned by another case.
func TestCrossCaseIsolation(t *testing.T) {
	db, close := setupDB()
	defer close()

	// case B owns an evidence
	orig := Evidence{ID: "shared", Name: "caseB", CaseID: "B"}
	assert.Nil(t, db.SaveEvidence("B", orig))

	t.Run("Get is case-scoped", func(t *testing.T) {
		_, err := db.GetEvidence("A", "shared")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		got, err := db.GetEvidence("B", "shared")
		assert.Nil(t, err)
		assert.Equal(t, "caseB", got.Name)
	})

	t.Run("Save cannot hijack another case's record", func(t *testing.T) {
		attack := Evidence{ID: "shared", Name: "attacker", CaseID: "A"}
		assert.ErrorIs(t, db.SaveEvidence("A", attack), ErrForeignCase)

		// case B's row is untouched
		got, err := db.GetEvidence("B", "shared")
		assert.Nil(t, err)
		assert.Equal(t, "caseB", got.Name)
		assert.Equal(t, "B", got.CaseID)
	})

	t.Run("Delete is case-scoped", func(t *testing.T) {
		assert.Nil(t, db.DeleteEvidence("A", "shared", "tester"))
		// still there
		_, err := db.GetEvidence("B", "shared")
		assert.Nil(t, err)

		// owner can delete
		assert.Nil(t, db.DeleteEvidence("B", "shared", "tester"))
		_, err = db.GetEvidence("B", "shared")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("same-case update still works", func(t *testing.T) {
		assert.Nil(t, db.SaveEvidence("C", Evidence{ID: "own", Name: "v1", CaseID: "C"}))
		assert.Nil(t, db.SaveEvidence("C", Evidence{ID: "own", Name: "v2", CaseID: "C"}))
		got, err := db.GetEvidence("C", "own")
		assert.Nil(t, err)
		assert.Equal(t, "v2", got.Name)
	})
}
