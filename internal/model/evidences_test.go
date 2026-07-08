package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvidenceLogsScopedByCase(t *testing.T) {
	db, close := setupDB()
	defer close()

	assert.Nil(t, db.SaveCase(Case{ID: "A", Name: "case A"}))
	assert.Nil(t, db.SaveCase(Case{ID: "B", Name: "case B"}))

	assert.Nil(t, db.SaveEvidenceLog("A", EvidenceLog{EvidenceID: "e1", Name: "e1.dd", User: "tester", Event: EvidenceLogUploaded}))
	assert.Nil(t, db.SaveEvidenceLog("B", EvidenceLog{EvidenceID: "e2", Name: "e2.dd", User: "tester", Event: EvidenceLogUploaded}))

	logsA, err := db.ListEvidenceLogs("A")
	assert.Nil(t, err)
	assert.Len(t, logsA, 1)
	assert.Equal(t, "e1", logsA[0].EvidenceID)

	logsB, err := db.ListEvidenceLogs("B")
	assert.Nil(t, err)
	assert.Len(t, logsB, 1)
	assert.Equal(t, "e2", logsB[0].EvidenceID)
}

func TestPurgeEvidenceLogsScopedToEvidence(t *testing.T) {
	db, close := setupDB()
	defer close()

	assert.Nil(t, db.SaveCase(Case{ID: "A", Name: "case A"}))
	assert.Nil(t, db.SaveEvidenceLog("A", EvidenceLog{EvidenceID: "e1", Name: "e1.dd", User: "tester", Event: EvidenceLogUploaded}))
	assert.Nil(t, db.SaveEvidenceLog("A", EvidenceLog{EvidenceID: "e1", Name: "e1.dd", User: "tester", Event: EvidenceLogDeleted}))
	assert.Nil(t, db.SaveEvidenceLog("A", EvidenceLog{EvidenceID: "e2", Name: "e2.dd", User: "tester", Event: EvidenceLogUploaded}))

	assert.Nil(t, db.PurgeEvidenceLogs("A", "e1"))

	logs, err := db.ListEvidenceLogs("A")
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "e2", logs[0].EvidenceID)
}

func TestForkCaseRemapsEvidenceLogs(t *testing.T) {
	db, close := setupDB()
	defer close()

	assert.Nil(t, db.SaveCase(Case{ID: "src", Name: "source case"}))
	assert.Nil(t, db.SaveEvidence("src", Evidence{ID: "evi01", Name: "image.dd", CaseID: "src"}))
	assert.Nil(t, db.SaveEvidenceLog("src", EvidenceLog{EvidenceID: "evi01", Name: "image.dd", User: "tester", Event: EvidenceLogUploaded, Details: "abc123"}))

	dst, err := db.ForkCase("src", Case{ID: "dst", Name: "forked case"})
	assert.Nil(t, err)

	evidences, err := db.ListEvidences(dst.ID)
	assert.Nil(t, err)
	assert.Len(t, evidences, 1)
	newEvidenceID := evidences[0].ID
	assert.NotEqual(t, "evi01", newEvidenceID)

	logs, err := db.ListEvidenceLogs(dst.ID)
	assert.Nil(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, newEvidenceID, logs[0].EvidenceID)
	assert.Equal(t, dst.ID, logs[0].CaseID)

	// the source case's own log row is untouched
	srcLogs, err := db.ListEvidenceLogs("src")
	assert.Nil(t, err)
	assert.Len(t, srcLogs, 1)
	assert.Equal(t, "evi01", srcLogs[0].EvidenceID)
}
