// Package cli implements the dagobert CLI subcommands (update, create-user, create-api-key).
package cli

import (
	"cmp"
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/internal/model"
)

// Update is the canonical "bring this instance current" command. It creates the
// database if missing, applies any pending migrations, and downloads/refreshes
// the MITRE ATT&CK data. It is idempotent: re-running it on an up-to-date
// instance is a no-op.
//
// --force ignores the skip-guards: it recovers a dirty database (re-running the
// failed migration) and re-downloads the MITRE data regardless of the sentinel.
func Update(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	if err := migrateDB(force); err != nil {
		return err
	}
	return updateMitre(force)
}

// migrateDB connects to the database (creating the file + parent dir if needed,
// see model.Connect) and applies pending migrations. With force, a dirty
// database is recovered by rolling back past the failed migration so Up re-runs
// it — the operator asserts they have fixed whatever caused the failure.
func migrateDB(force bool) error {
	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("Connecting to database", "url", dburl)
	store, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	slog.Info("Loading database migrations")
	m, err := store.NewMigrate()
	if err != nil {
		return err
	}

	// --------------------------------------
	// Dirty recovery (--force)
	// --------------------------------------
	if force {
		version, dirty, verr := m.Version()
		if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
			return verr
		}

		switch {
		case !dirty:
			slog.Info("Database is not dirty, --force has no effect on migrations")
		default:
			// Roll the recorded version back past the failed migration so that
			// Up re-runs it. By passing --force the operator asserts they have
			// fixed whatever caused migration to fail.
			reset := int(version) - 1
			if reset < 1 {
				reset = -1 // migrate.NilVersion: re-run from the first migration
			}
			slog.Warn("Forcing dirty database to re-run failed migration", "dirty_version", version, "reset_to", reset)
			if err := m.Force(reset); err != nil {
				return err
			}
		}
	}

	slog.Info("Applying database migrations")
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	if dirty {
		slog.Warn("Database model dirty", "version", v)
	} else {
		slog.Info("Database model current", "version", v)
	}
	return nil
}
