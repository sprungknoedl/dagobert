package model

import (
	"errors"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// SchemaState classifies the database schema relative to the migrations
// embedded in this build.
type SchemaState int

const (
	SchemaCurrent SchemaState = iota // database matches this build
	SchemaBehind                     // database is older; migrations are pending
	SchemaAhead                      // database is newer; this binary is stale
	SchemaDirty                      // a previous migration failed partway
)

// SchemaStatus reports how the connected database compares to the migrations
// embedded in this build.
type SchemaStatus struct {
	State   SchemaState
	Current uint // database version; 0 when nothing has been applied yet
	Dirty   bool
	Latest  uint // newest migration embedded in this build
}

// NewMigrate binds the embedded migrations to the store's connection.
//
// The caller must not Close() the returned instance when the store is shared
// with a running server: closing it would close the underlying connection.
func (store *Store) NewMigrate() (*migrate.Migrate, error) {
	driver, err := sqlite.WithInstance(store.RawConn, &sqlite.Config{})
	if err != nil {
		return nil, err
	}

	source, err := iofs.New(Migrations, "migrations")
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("iofs", source, "sqlite", driver)
}

// CheckSchema reports how the database schema relates to this build without
// modifying anything.
func (store *Store) CheckSchema() (SchemaStatus, error) {
	latest, err := latestMigration()
	if err != nil {
		return SchemaStatus{}, err
	}

	m, err := store.NewMigrate()
	if err != nil {
		return SchemaStatus{}, err
	}

	current, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		// fresh database, nothing applied yet
		return SchemaStatus{State: SchemaBehind, Current: 0, Latest: latest}, nil
	}
	if err != nil {
		return SchemaStatus{}, err
	}

	status := SchemaStatus{Current: current, Dirty: dirty, Latest: latest}
	switch {
	case dirty:
		status.State = SchemaDirty
	case current < latest:
		status.State = SchemaBehind
	case current > latest:
		status.State = SchemaAhead
	default:
		status.State = SchemaCurrent
	}
	return status, nil
}

// SchemaVersion reports the highest applied migration this build understands,
// i.e. the schema version stamped into (and checked against) a case archive.
func SchemaVersion() (uint, error) {
	return latestMigration()
}

// latestMigration returns the highest version among the embedded migration
// files (their numeric filename prefix).
func latestMigration() (uint, error) {
	entries, err := Migrations.ReadDir("migrations")
	if err != nil {
		return 0, err
	}

	var latest uint
	for _, e := range entries {
		name := e.Name()
		i := strings.IndexByte(name, '_')
		if i <= 0 {
			continue
		}
		n, err := strconv.ParseUint(name[:i], 10, 64)
		if err != nil {
			continue
		}
		if uint(n) > latest {
			latest = uint(n)
		}
	}
	return latest, nil
}
