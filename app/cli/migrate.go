package cli

import (
	"cmp"
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/model"
)

func Migrate(cmd *cobra.Command, args []string) error {
	// --------------------------------------
	// Database
	// --------------------------------------
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
	if force, _ := cmd.Flags().GetBool("force"); force {
		version, dirty, verr := m.Version()
		if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
			return verr
		}

		switch {
		case !dirty:
			slog.Info("Database is not dirty, --force has no effect")
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
