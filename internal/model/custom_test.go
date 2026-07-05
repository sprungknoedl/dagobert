package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureCustomAttribute(t *testing.T) {
	db, close := setupDB()
	defer close()

	find := func(label string) (CustomAttribute, bool) {
		list, err := db.ListCustomAttributes()
		assert.Nil(t, err)
		for _, ca := range list {
			if ca.Entity == "Indicator" && ca.Label == label {
				return ca, true
			}
		}
		return CustomAttribute{}, false
	}

	t.Run("creates a missing definition", func(t *testing.T) {
		err := db.EnsureCustomAttribute("Indicator", "MISP Verdict", "select", Strings{"malicious", "clean"}, 5)
		assert.Nil(t, err)

		ca, ok := find("MISP Verdict")
		assert.True(t, ok)
		assert.Equal(t, "select", ca.Type)
		assert.Equal(t, Strings{"malicious", "clean"}, ca.Options)
		assert.Equal(t, 5, ca.Rank)
	})

	t.Run("is idempotent and does not overwrite an admin-tweaked row", func(t *testing.T) {
		// admin tweaks Rank/Options after the first creation
		ca, _ := find("MISP Verdict")
		ca.Rank = 99
		ca.Options = Strings{"custom"}
		assert.Nil(t, db.SaveCustomAttribute(ca))

		// a restart re-ensures with the original values, but must not clobber
		err := db.EnsureCustomAttribute("Indicator", "MISP Verdict", "select", Strings{"malicious", "clean"}, 5)
		assert.Nil(t, err)

		got, ok := find("MISP Verdict")
		assert.True(t, ok)
		assert.Equal(t, 99, got.Rank)
		assert.Equal(t, Strings{"custom"}, got.Options)

		// still exactly one row
		list, _ := db.ListCustomAttributes()
		count := 0
		for _, c := range list {
			if c.Entity == "Indicator" && c.Label == "MISP Verdict" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})
}
