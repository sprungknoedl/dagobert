package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerdictSeverity(t *testing.T) {
	tests := []struct {
		v    string
		rank int
		ok   bool
	}{
		{"malicious", 3, true},
		{"suspicious", 2, true},
		{"clean", 1, true},
		{"unknown", 0, true},
		{"Malicious", 3, true},
		{"SUSPICIOUS", 2, true},
		{"Clean", 1, true},
		{"UNKNOWN", 0, true},
		{"harmless", 0, false},
		{"not found", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		rank, ok := VerdictSeverity(tt.v)
		assert.Equal(t, tt.ok, ok, "ok for %q", tt.v)
		if tt.ok {
			assert.Equal(t, tt.rank, rank, "rank for %q", tt.v)
		}
	}
}

// TestSetEnrichmentDeleted checks that a job finishing after its object was
// deleted does not recreate an orphaned enrichment row.
func TestSetEnrichmentDeleted(t *testing.T) {
	db, closer := setupDB()
	defer closer()
	require.Nil(t, db.SaveCase(Case{ID: "case01", Name: "Test case"}))
	require.Nil(t, db.SaveIndicator("case01", Indicator{ID: "ind01", Type: "IP", Value: "198.51.100.7"}, false))
	e := Enrichment{CaseID: "case01", ObjectType: "Indicator", ObjectID: "ind01", Module: "VirusTotal"}

	require.Nil(t, db.SetEnrichment(e))
	require.Nil(t, db.DeleteIndicator("case01", "ind01"))
	require.Nil(t, db.SetEnrichment(e))

	list, err := db.ListEnrichments("Indicator", "ind01")
	require.Nil(t, err)
	assert.Empty(t, list)
}
