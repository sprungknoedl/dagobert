package model

import (
	"testing"

	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/stretchr/testify/assert"
)

func TestSetIndicatorCustom(t *testing.T) {
	db, close := setupDB()
	defer close()

	kase := Case{ID: fp.Random(10), Name: "Test Case"}
	assert.Nil(t, db.SaveCase(kase))

	ind := Indicator{ID: fp.Random(10), CaseID: kase.ID, Type: "IP", Value: "185.220.101.5"}
	assert.Nil(t, db.SaveIndicator(kase.ID, ind, true))

	get := func() Custom {
		obj, err := db.GetIndicator(kase.ID, ind.ID)
		assert.Nil(t, err)
		return obj.Custom
	}

	t.Run("a multi-field write merges into an empty-seeded column", func(t *testing.T) {
		err := db.SetIndicatorCustom(kase.ID, ind.ID, map[string]string{
			"MISP Enrichment": "Verdict: malicious",
			"MISP Verdict":    "malicious",
		})
		assert.Nil(t, err)
		assert.Equal(t, Custom{
			"MISP Enrichment": "Verdict: malicious",
			"MISP Verdict":    "malicious",
		}, get())
	})

	t.Run("a second module writing different keys coexists", func(t *testing.T) {
		err := db.SetIndicatorCustom(kase.ID, ind.ID, map[string]string{
			"VirusTotal Verdict": "clean",
		})
		assert.Nil(t, err)
		assert.Equal(t, Custom{
			"MISP Enrichment":    "Verdict: malicious",
			"MISP Verdict":       "malicious",
			"VirusTotal Verdict": "clean",
		}, get())
	})

	t.Run("a same-key write overwrites", func(t *testing.T) {
		err := db.SetIndicatorCustom(kase.ID, ind.ID, map[string]string{
			"MISP Verdict": "suspicious",
		})
		assert.Nil(t, err)
		assert.Equal(t, "suspicious", get()["MISP Verdict"])
	})

	t.Run("an empty field is not written as a key", func(t *testing.T) {
		err := db.SetIndicatorCustom(kase.ID, ind.ID, map[string]string{
			"MISP Link": "",
		})
		assert.Nil(t, err)
		_, ok := get()["MISP Link"]
		assert.False(t, ok)
	})

	t.Run("a label with spaces round-trips through the JSON path", func(t *testing.T) {
		err := db.SetIndicatorCustom(kase.ID, ind.ID, map[string]string{
			"MISP Enrichment": "Sightings: 12 across 3 events",
		})
		assert.Nil(t, err)
		assert.Equal(t, "Sightings: 12 across 3 events", get()["MISP Enrichment"])
	})
}
