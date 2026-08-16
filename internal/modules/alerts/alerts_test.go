package alerts

import (
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDB(t *testing.T) *model.Store {
	db, err := model.Connect(":memory:")
	require.Nil(t, err)
	t.Cleanup(func() { db.RawConn.Close() })

	source, _ := iofs.New(model.Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	require.Nil(t, m.Up())
	return db
}

func TestSupports(t *testing.T) {
	m := &Module{}

	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{"jsonl passes", model.Evidence{Name: "alerts.jsonl"}, true},
		{"evtx rejected", model.Evidence{Name: "Security.evtx"}, false},
		{"no extension rejected", model.Evidence{Name: "README"}, false},
		{"non-evidence rejected", model.Indicator{Type: "Hash"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Supports(tc.obj))
		})
	}
}

const wellFormedLine = `{"datetime":"2019-03-19T00:02:04.319945+00:00","timestamp_desc":"Event time","message":"Rare Schtasks Creations","level":"high","tags":["attack.execution","attack.t1059.001"]}`

func TestImportAlerts(t *testing.T) {
	t.Run("well-formed line with tags and timestamp_desc is imported", func(t *testing.T) {
		db := setupDB(t)
		kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
		require.Nil(t, db.SaveCase(kase))

		imported, filtered, skipped, err := importAlerts(db, kase.ID, "alerts.jsonl", strings.NewReader(wellFormedLine+"\n"), "all")
		require.Nil(t, err)
		assert.Equal(t, 1, imported)
		assert.Equal(t, 0, filtered)
		assert.Equal(t, 0, skipped)

		events, err := db.ListEvents(kase.ID)
		require.Nil(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "Event time Rare Schtasks Creations", events[0].Event)
		assert.Contains(t, events[0].Techniques, "T1059.001")
		assert.Equal(t, "alerts.jsonl", events[0].Source)
	})

	t.Run("line missing datetime is skipped", func(t *testing.T) {
		db := setupDB(t)
		kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
		require.Nil(t, db.SaveCase(kase))

		line := `{"message":"no datetime here","level":"high"}`
		imported, filtered, skipped, err := importAlerts(db, kase.ID, "alerts.jsonl", strings.NewReader(line+"\n"), "all")
		require.Nil(t, err)
		assert.Equal(t, 0, imported)
		assert.Equal(t, 0, filtered)
		assert.Equal(t, 1, skipped)

		events, err := db.ListEvents(kase.ID)
		require.Nil(t, err)
		assert.Len(t, events, 0)
	})

	t.Run("severity filter excludes below the bar and admits at/above it", func(t *testing.T) {
		db := setupDB(t)
		kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
		require.Nil(t, db.SaveCase(kase))

		medium := `{"datetime":"2019-03-19T00:02:04.319945+00:00","message":"medium alert","level":"medium"}`
		critical := `{"datetime":"2019-03-19T00:02:05.319945+00:00","message":"critical alert","level":"critical"}`
		lines := strings.Join([]string{medium, critical}, "\n")

		imported, filtered, skipped, err := importAlerts(db, kase.ID, "alerts.jsonl", strings.NewReader(lines), "high")
		require.Nil(t, err)
		assert.Equal(t, 1, imported)
		assert.Equal(t, 1, filtered)
		assert.Equal(t, 0, skipped)

		events, err := db.ListEvents(kase.ID)
		require.Nil(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "critical alert", events[0].Event)
	})

	t.Run("no level field is filtered out when a severity filter is active", func(t *testing.T) {
		db := setupDB(t)
		kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
		require.Nil(t, db.SaveCase(kase))

		line := `{"datetime":"2019-03-19T00:02:04.319945+00:00","message":"no level here"}`
		imported, filtered, skipped, err := importAlerts(db, kase.ID, "alerts.jsonl", strings.NewReader(line+"\n"), "critical")
		require.Nil(t, err)
		assert.Equal(t, 0, imported)
		assert.Equal(t, 1, filtered)
		assert.Equal(t, 0, skipped)
	})

	t.Run("rerunning on the same file does not duplicate or clobber an analyst edit", func(t *testing.T) {
		db := setupDB(t)
		kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
		require.Nil(t, db.SaveCase(kase))

		_, _, _, err := importAlerts(db, kase.ID, "alerts.jsonl", strings.NewReader(wellFormedLine+"\n"), "all")
		require.Nil(t, err)

		events, err := db.ListEvents(kase.ID)
		require.Nil(t, err)
		require.Len(t, events, 1)

		edited := events[0]
		edited.Event = "analyst-edited text"
		require.Nil(t, db.SaveEvent(kase.ID, edited, true))

		_, _, _, err = importAlerts(db, kase.ID, "alerts.jsonl", strings.NewReader(wellFormedLine+"\n"), "all")
		require.Nil(t, err)

		events, err = db.ListEvents(kase.ID)
		require.Nil(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "analyst-edited text", events[0].Event)
	})
}
