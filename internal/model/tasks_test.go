package model

import (
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrateTaskOwner checks the 046_task_owner_user migration: each existing
// owner string is matched case-insensitively against a user's name, email, or
// login, and a non-matching owner leaves the task unassigned.
func TestMigrateTaskOwner(t *testing.T) {
	db, err := Connect(":memory:")
	require.Nil(t, err)
	t.Cleanup(func() { db.RawConn.Close() })

	source, _ := iofs.New(Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	require.Nil(t, m.Migrate(45))

	require.Nil(t, db.SaveUser(User{ID: "u1", Name: "Alice", Login: "alice-login", Email: "alice@example.com"}))
	require.Nil(t, db.SaveUser(User{ID: "u2", Name: "Bob", Login: "bob-login", Email: "bob@example.com"}))
	require.Nil(t, db.SaveUser(User{ID: "u3", Name: "Carol", Login: "carol", Email: "carol@example.com"}))
	require.Nil(t, db.SaveCase(Case{ID: "case01", Name: "Test"}))

	tasks := []struct{ id, owner string }{
		{"t1", "Alice"},           // matches user name, different case
		{"t2", "bob@example.com"}, // matches user's email
		{"t3", "carol"},           // matches login
		{"t4", "Client IT"},       // matches nothing
	}
	for _, tc := range tasks {
		require.Nil(t, db.DB.Exec(
			`INSERT INTO tasks (id, case_id, type, task, done, owner, date_due) VALUES (?, ?, 'Analysis', 'x', 0, ?, ?)`,
			tc.id, "case01", tc.owner, time.Now()).Error)
	}

	require.Nil(t, m.Migrate(46))

	want := map[string]*string{"t1": new("u1"), "t2": new("u2"), "t3": new("u3"), "t4": nil}
	for id, wantOwner := range want {
		var got struct{ OwnerID *string }
		require.Nil(t, db.DB.Raw("SELECT owner_id FROM tasks WHERE id = ?", id).Scan(&got).Error)
		assert.Equal(t, wantOwner, got.OwnerID, "task %s", id)
	}
}

func TestDeleteUserClearsTaskOwner(t *testing.T) {
	db := setupDBWithForeignKeys(t)

	user := User{ID: "userA", Name: "Alice"}
	require.Nil(t, db.SaveUser(user))
	require.Nil(t, db.SaveCase(Case{ID: "case01", Name: "Test"}))

	task := Task{ID: "task01", CaseID: "case01", Type: "Analysis", Task: "Review", OwnerID: &user.ID}
	require.Nil(t, db.SaveTask("case01", task))

	require.Nil(t, db.DeleteUser(user.ID))

	got, err := db.GetTask("case01", task.ID)
	require.Nil(t, err)
	assert.Nil(t, got.OwnerID)
}

func TestCloneCaseContentsClearsTaskOwner(t *testing.T) {
	db, close := setupDB()
	defer close()

	user := User{ID: "userA", Name: "Alice"}
	require.Nil(t, db.SaveUser(user))

	src := Case{ID: "case01", Name: "Source"}
	require.Nil(t, db.SaveCase(src))
	task := Task{ID: "task01", CaseID: src.ID, Type: "Analysis", Task: "Review", OwnerID: &user.ID}
	require.Nil(t, db.SaveTask(src.ID, task))

	dst := Case{ID: "case02", Name: "Clone"}
	_, err := db.CloneCaseContents(src.ID, dst)
	require.Nil(t, err)

	got, err := db.ListTasks(dst.ID)
	require.Nil(t, err)
	require.Len(t, got, 1)
	assert.Nil(t, got[0].OwnerID)
}
