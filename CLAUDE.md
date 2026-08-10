# Dagobert

A collaborative incident-response investigation tool, written in Go. It's one self-contained
binary (`dagobert`) that serves a web app — no other services required. Pages are rendered on the
server as templ fragments, using Unpoly: no single-page app, no JavaScript build step.

## Commands
- Check any change with `make check`. CI runs the exact same command, so nothing is done until it
  passes.
- Run `make validate-exports` after changing the export mapping in
  `internal/handler/indicators.go`. This needs network access plus python3/xmllint, so it's not
  included in `make check`.

## Architecture
- `Object` (`internal/model/`) wraps Evidence, Indicator, and Malware into one type, and tracks
  which one it actually is.
- Views (`internal/views/`): get the current user and their access rights (ACL) from middleware.
- Worker (`internal/modules/`): a pool running inside the app checks the jobs table
  (Scheduled → Running → Success/Failed) and runs external commands set by `MODULE_*` env vars.
  This package also has the automation-rules engine, which runs expr-lang conditions when a
  record is created or updated.
- ACL: role-based access control via Casbin. Roles are Administrator, User, and Read-Only, set
  per case under `/cases/{cid}/acl`.
- Frontend behavior (`internal/assets/dagobert.js`): each behavior is one `up.compiler()`, run by
  Unpoly after every page-fragment update (on first load, and when overlays/drawers render). The
  server closes overlays and shows toast messages using the `X-Up-Accept-Layer` header (see
  `RedirectAfterSave` in `internal/handler/handler.go`). A 422 response shows form errors inline.
- `/mcp` (`internal/handler/mcp.go`): a read-only MCP endpoint that exposes case data as tools. It
  runs separately from the main ACL-gated `Handler`.

## Conventions
- YAGNI and KISS: build what the request needs.
- To add a migration: create `NNN_name.up.sql` and `NNN_name.down.sql` in
  `internal/model/migrations/`. Never change an existing migration.
- Only validate at the boundaries: user input and external APIs. Use `pkg/valid`'s fluent style:
  `v.Check(cond, field, msg)`. Enum values are checked against value lists stored in the database
  (which users can customize), never against Go constants.
- For CSV import, use `ImportCSV()` in `internal/handler/handler.go` with a closure that processes
  each row. Use `cmp.Or()` to set default values for fields.
- User data lives in `data/`. System data that gets downloaded (MITRE, Chainsaw/Hayabusa rules)
  lives in `external/`, and is filled in by running `dagobert update`. Temporary files (uploads in
  progress, build-tool caches) live in `tmp/`.
- If you're changing anything in the UI: `DESIGN.md` has the visual design system (tokens and
  named rules), and `PRODUCT.md` explains who this is for and what's actually shipped.

## Gotchas
- Never edit `*_templ.go` files by hand — run `go tool templ generate` instead (this also happens
  automatically as part of `make build-go` / `make check`).
- `internal/assets/dagobert.css` is a compiled file (it's committed to git and loaded with
  `go:embed`). To change it, edit `internal/frontend/dagobert.css` instead, then run
  `make build-web`.
- The MITRE ATT&CK JSON data isn't stored in git — run `./dagobert update` to download it before
  working on features that need it.
- Config: every environment variable (OIDC, `MODULE_*`, `TIMESKETCH_*`, `DAGOBERT_WORKERS`) is
  listed in `dagobert.env.example`.
