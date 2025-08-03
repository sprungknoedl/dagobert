package cli

import (
	"cmp"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/model"
)

func Migrate(cmd *cobra.Command, args []string) error {
	// --------------------------------------
	// Database
	// --------------------------------------
	dburl := cmp.Or(os.Getenv("DB_URL"), "file:files/dagobert.db?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)")
	slog.Info("Connecting to database", "url", dburl)
	store, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	db, err := sqlite.WithInstance(store.RawConn, &sqlite.Config{})
	if err != nil {
		return err
	}

	slog.Info("Loading database migrations")
	source, err := iofs.New(model.Migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", db)
	if err != nil {
		return err
	}

	slog.Info("Applying database migrations")
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	v, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	if dirty {
		slog.Info("Database model migrated", "version", v)
	} else {
		slog.Info("Database model current", "version", v)
	}
	return nil
}
