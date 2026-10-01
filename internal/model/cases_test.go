package model

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDBWithForeignKeys is setupDB plus foreign-key enforcement. Production
// connections (DefaultUrl) always enable it; setupDB's plain ":memory:" DSN
// doesn't, so the case_assignees cascade deletes this test exercises need
// their own setup. SetMaxOpenConns(1) keeps every query on the one
// connection the PRAGMA (a per-connection SQLite setting) was set on.
func setupDBWithForeignKeys(t *testing.T) *Store {
	t.Helper()
	db, err := Connect(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.RawConn.Close() })
	db.RawConn.SetMaxOpenConns(1)

	if err := db.DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}

	source, _ := iofs.New(Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}

	return db
}

func TestSaveCaseAssignees(t *testing.T) {
	db := setupDBWithForeignKeys(t)

	userA := User{ID: "userA", Name: "Alice", Login: "alice"}
	userB := User{ID: "userB", Name: "Bob", Login: "bob"}
	require.Nil(t, db.SaveUser(userA))
	require.Nil(t, db.SaveUser(userB))

	kase := Case{ID: "case01", Name: "Test case"}
	require.Nil(t, db.SaveCase(kase))

	t.Run("replaces assignees on save, does not accumulate", func(t *testing.T) {
		kase.Assignees = []User{{ID: userA.ID}, {ID: userB.ID}}
		require.Nil(t, db.SaveCase(kase))

		got, err := db.GetCaseWithAssignees(kase.ID)
		require.Nil(t, err)
		assert.ElementsMatch(t, []string{userA.ID, userB.ID},
			fp.Apply(got.Assignees, func(u User) string { return u.ID }))

		kase.Assignees = []User{{ID: userB.ID}}
		require.Nil(t, db.SaveCase(kase))

		got, err = db.GetCaseWithAssignees(kase.ID)
		require.Nil(t, err)
		require.Len(t, got.Assignees, 1)
		assert.Equal(t, userB.ID, got.Assignees[0].ID)
	})

	t.Run("deleting a user removes their assignment, case keeps the rest", func(t *testing.T) {
		require.Nil(t, db.DeleteUser(userB.ID))

		got, err := db.GetCaseWithAssignees(kase.ID)
		require.Nil(t, err)
		assert.Empty(t, got.Assignees)
	})

	t.Run("deleting a case removes its assignment rows", func(t *testing.T) {
		require.Nil(t, db.SaveUser(userB)) // subtest above deleted it
		kase.Assignees = []User{{ID: userA.ID}, {ID: userB.ID}}
		require.Nil(t, db.SaveCase(kase))

		require.Nil(t, db.DeleteCase(kase.ID))

		var count int64
		require.Nil(t, db.DB.Table("case_assignees").Where("case_id = ?", kase.ID).Count(&count).Error)
		assert.Zero(t, count)
	})
}

func TestForkCaseDropsAssignees(t *testing.T) {
	db, close := setupDB()
	defer close()

	user := User{ID: "userA", Name: "Alice"}
	require.Nil(t, db.SaveUser(user))

	src := Case{ID: "case01", Name: "Source", Assignees: []User{{ID: user.ID}}}
	require.Nil(t, db.SaveCase(src))

	arch, err := db.ExportCaseArchive(src.ID)
	require.Nil(t, err)
	assert.Empty(t, arch.Case.Assignees)

	dst := Case{ID: "case02", Name: "Forked"}
	_, err = db.ForkCase(src.ID, dst)
	require.Nil(t, err)

	got, err := db.GetCaseWithAssignees(dst.ID)
	require.Nil(t, err)
	assert.Empty(t, got.Assignees)
}
