// Package cli implements the dagobert CLI subcommands (update, create-user, create-api-key).
package cli

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

// Update is the canonical "bring this instance current" command. It creates the
// database if missing, applies any pending migrations, downloads/refreshes the
// MITRE ATT&CK data, and fetches any other registered module's pinned vendor
// data (Sigma rules, mapping files, etc.) via its AssetUpdater hook. It is
// idempotent: re-running it on an up-to-date instance is a no-op.
//
// --force ignores the skip-guards: it recovers a dirty database (re-running the
// failed migration) and re-downloads the MITRE data regardless of the sentinel.
func Update(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	if err := migrateDB(force); err != nil {
		return err
	}
	if err := updateMitre(force); err != nil {
		return err
	}

	ts := timesketch.NewClient(timesketch.Config{
		URL:           os.Getenv("TIMESKETCH_URL"),
		Username:      os.Getenv("TIMESKETCH_USER"),
		Password:      os.Getenv("TIMESKETCH_PASS"),
		SkipVerifyTLS: os.Getenv("TIMESKETCH_SKIP_VERIFY_TLS") == "true",
	})
	modules.Register(ts)
	return modules.UpdateAssets(context.Background())
}

// migrateDB connects to the database (creating the file + parent dir if needed,
// see model.Connect) and applies pending migrations. With force, a dirty
// database is recovered by rolling back past the failed migration so Up re-runs
// it — the operator asserts they have fixed whatever caused the failure.
func migrateDB(force bool) error {
	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("connecting to database", "url", dburl)
	store, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	slog.Info("loading database migrations")
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
			slog.Info("database is not dirty, --force has no effect on migrations")
		default:
			// Roll the recorded version back past the failed migration so that
			// Up re-runs it. By passing --force the operator asserts they have
			// fixed whatever caused migration to fail.
			reset := int(version) - 1
			if reset < 1 {
				reset = -1 // migrate.NilVersion: re-run from the first migration
			}
			slog.Warn("forcing dirty database to re-run failed migration", "dirty_version", version, "reset_to", reset)
			if err := m.Force(reset); err != nil {
				return err
			}
		}
	}

	slog.Info("applying database migrations")
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	if dirty {
		slog.Warn("database model dirty", "version", v)
	} else {
		slog.Info("database model current", "version", v)
	}
	return nil
}
