package model

import (
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
)

// Chronological order, but written with mixed offsets and fraction widths so
// that RFC3339Nano text order differs from time order.
var mixedTimes = []string{
	"2025-03-01T12:00:05+02:00",        // 10:00:05Z
	"2025-03-01T10:00:05Z",             // 10:00:05Z, same instant
	"2025-03-01T10:00:05.5Z",           // sorted before "…:05Z" as RFC3339Nano text
	"2025-03-01T06:00:06.25-04:00",     // 10:00:06.25Z
	"2025-03-01T10:00:06.250000001Z",   // nanosecond later
	"2025-03-01T11:00:07.123456+01:00", // 10:00:07.123456Z
}

// TestTimeValue checks that stored times are UTC with a fixed-width fraction,
// so their text order is chronological, and that they scan back to the same
// instant.
func TestTimeValue(t *testing.T) {
	prev := ""
	for i, s := range mixedTimes {
		in, err := time.Parse(time.RFC3339Nano, s)
		assert.Nil(t, err)

		v, err := Time(in).Value()
		assert.Nil(t, err)
		str := v.(string)
		assert.Len(t, str, len("2006-01-02T15:04:05.000000000Z"), s)
		assert.LessOrEqual(t, prev, str, "row %d", i)
		prev = str

		var out Time
		assert.Nil(t, out.Scan(str))
		assert.True(t, in.Equal(time.Time(out)), s)
	}
}

// TestListEventsOrder saves events in reverse with mixed offsets and
// fractional seconds, and expects the timeline back in chronological order.
func TestListEventsOrder(t *testing.T) {
	db, close := setupDB()
	defer close()

	assert.Nil(t, db.SaveCase(Case{ID: "A", Name: "case A"}))
	for i := len(mixedTimes) - 1; i >= 0; i-- {
		tm, err := time.Parse(time.RFC3339Nano, mixedTimes[i])
		assert.Nil(t, err)
		assert.Nil(t, db.SaveEvent("A", Event{ID: mixedTimes[i], Time: Time(tm)}, false))
	}

	events, err := db.ListEvents("A")
	assert.Nil(t, err)
	assert.Len(t, events, len(mixedTimes))
	for i := 1; i < len(events); i++ {
		assert.False(t, time.Time(events[i].Time).Before(time.Time(events[i-1].Time)), "event %d (%s) before %s", i, events[i].ID, events[i-1].ID)
	}
}

// TestUTCTimesMigration writes rows in the old RFC3339Nano format before
// migration 047 and expects them rewritten to the new layout without losing
// sub-millisecond precision.
func TestUTCTimesMigration(t *testing.T) {
	db, err := Connect(":memory:")
	assert.Nil(t, err)
	defer db.RawConn.Close()
	db.RawConn.SetMaxOpenConns(1)

	source, _ := iofs.New(Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	assert.Nil(t, err)
	assert.Nil(t, m.Migrate(46))

	assert.Nil(t, db.DB.Exec("INSERT INTO cases (id, name, closed, classification, severity, outcome, summary) VALUES ('A', 'case A', false, '', '', '', '')").Error)
	for i := len(mixedTimes) - 1; i >= 0; i-- {
		assert.Nil(t, db.DB.Exec("INSERT INTO events (id, case_id, time, type, event, raw, flagged) VALUES (?, 'A', ?, '', '', '', false)", mixedTimes[i], mixedTimes[i]).Error)
	}
	assert.Nil(t, db.DB.Exec("INSERT INTO users (id, name, login, email, role, last_login) VALUES ('u1', 'u1', 'u1', 'u1', '', NULL)").Error)

	assert.Nil(t, m.Up())

	// || '' keeps the driver from parsing DATETIME text into time.Time.
	var got []string
	assert.Nil(t, db.DB.Raw("SELECT time || '' FROM events ORDER BY time, rowid").Scan(&got).Error)
	assert.Equal(t, []string{
		"2025-03-01T10:00:05.000000000Z",
		"2025-03-01T10:00:05.000000000Z",
		"2025-03-01T10:00:05.500000000Z",
		"2025-03-01T10:00:06.250000000Z",
		"2025-03-01T10:00:06.250000001Z",
		"2025-03-01T10:00:07.123456000Z",
	}, got)

	var login *string
	assert.Nil(t, db.DB.Raw("SELECT last_login || '' FROM users WHERE id = 'u1'").Scan(&login).Error)
	assert.Nil(t, login)
}
