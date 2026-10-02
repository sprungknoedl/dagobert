package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddFromFSRerun registers the same module output twice, as a module
// re-run does after overwriting its output file. The second call must update
// the existing row (same ID, new hash, analyst notes kept) instead of failing
// on the per-case unique name.
func TestAddFromFSRerun(t *testing.T) {
	t.Chdir(t.TempDir())
	db, err := model.Connect(":memory:")
	require.Nil(t, err)
	t.Cleanup(func() { db.RawConn.Close() })
	source, _ := iofs.New(model.Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	require.Nil(t, m.Up())

	OnEvidenceAdded = func(*model.Store, model.Evidence) {}
	t.Cleanup(func() { OnEvidenceAdded = nil })

	require.Nil(t, db.SaveCase(model.Case{ID: "case01", Name: "Test Case"}))
	obj := model.Evidence{CaseID: "case01", Type: "Logs", Name: "Security.evtx.hayabusa.jsonl", Notes: "module-hayabusa"}
	path := Filepath(obj)
	require.Nil(t, os.MkdirAll(filepath.Dir(path), 0755))

	require.Nil(t, os.WriteFile(path, []byte("first\n"), 0644))
	require.Nil(t, AddFromFS(db, obj, "Hayabusa"))
	first, err := db.ListEvidences("case01")
	require.Nil(t, err)
	require.Len(t, first, 1)

	edited := first[0]
	edited.Notes = "analyst notes"
	require.Nil(t, db.SaveEvidence("case01", edited))

	require.Nil(t, os.WriteFile(path, []byte("second run\n"), 0644))
	require.Nil(t, AddFromFS(db, obj, "Hayabusa"))
	second, err := db.ListEvidences("case01")
	require.Nil(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, first[0].ID, second[0].ID)
	assert.NotEqual(t, first[0].Hash, second[0].Hash)
	assert.Equal(t, int64(len("second run\n")), second[0].Size)
	assert.Equal(t, "analyst notes", second[0].Notes)
}
