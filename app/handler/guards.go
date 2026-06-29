package handler

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/sprungknoedl/dagobert/app/model"
)

// Startup guards: preflight checks run before the server begins serving. Each
// verifies one precondition and, when it fails, prints a human-readable
// explanation of how to fix it (typically `dagobert update`) and exits. They
// never modify state.

// guardSchema refuses to start the server when the database schema does not
// match the migrations embedded in this build. It prints a human-readable
// explanation and exits; it never modifies the database.
func guardSchema(db *model.Store) {
	status, err := db.CheckSchema()
	if err != nil {
		slog.Error("Failed to check database schema", "err", err)
		os.Exit(1)
	}

	switch status.State {
	case model.SchemaCurrent:
		return
	case model.SchemaBehind:
		fmt.Fprintf(os.Stderr, schemaBehind, status.Current, status.Latest, status.Latest-status.Current)
	case model.SchemaDirty:
		fmt.Fprintf(os.Stderr, schemaDirty, status.Current)
	case model.SchemaAhead:
		fmt.Fprintf(os.Stderr, schemaAhead, status.Current, status.Latest)
	}
	os.Exit(1)
}

// guardMitre refuses to start the server when the MITRE ATT&CK data is missing.
// The data is required (it is woven into events, indicators and the ATT&CK
// view), so failing fast with a clear pointer to `dagobert update` beats a
// server that starts and breaks confusingly at request time.
func guardMitre(paths ...string) {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			fmt.Fprint(os.Stderr, mitreMissing)
			os.Exit(1)
		}
	}
}

const mitreMissing = `
✗ MITRE ATT&CK data not found.

  dagobert needs the ATT&CK knowledge base to start. To download it, run:

      dagobert update
      # docker:  docker compose run --rm app update

  Then start the server again.
`

const schemaBehind = `
✗ Database schema is out of date.

  Your database is at migration   %d
  This dagobert build expects     %d  (%d migration(s) pending)

  The database was not changed. To apply the pending migration(s), run:

      dagobert update
      # docker:  docker compose run --rm app update

  Then start the server again. Tip: back up files/dagobert.db first.
`

const schemaDirty = `
✗ Database is in a dirty state — a previous migration failed at version %d.

  dagobert will not start until this is resolved, to avoid corrupting case data.
  Restore your most recent backup of files/dagobert.db, or once you have
  confirmed the schema by hand, recover with:

      dagobert update --force
      # docker:  docker compose run --rm app update --force
`

const schemaAhead = `
✗ This dagobert build is older than your database.

  Database is at migration   %d
  This build understands      %d

  Use a newer dagobert build, or restore a backup from before the upgrade.
  Running with a mismatched schema risks corrupting case data.
`
