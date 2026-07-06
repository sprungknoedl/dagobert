package handler

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/internal/auth"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/pkg/attck"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/public"
)

func Run(cmd *cobra.Command, args []string) {
	// --------------------------------------
	// Database
	// --------------------------------------
	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Debug("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		slog.Error("Failed to connect to database", "err", err)
		return
	}

	// refuse to serve against a schema that does not match this build; running
	// out-of-date or partially-migrated would fail confusingly at request time
	guardSchema(db)

	// --------------------------------------
	// Authorization
	// --------------------------------------
	slog.Debug("Creating casbin acl model")
	acl := auth.NewACL(db)

	// --------------------------------------
	// Authentication
	// --------------------------------------
	initSession(db.RawConn)
	a, err := auth.New(db, Session)
	if err != nil {
		slog.Error("Failed to initialize auth", "err", err)
		return
	}

	// --------------------------------------
	// MITRE ATT&CK
	// --------------------------------------
	// system data lives outside files/, which holds user data only (and is
	// shadowed by the volume mount in Docker). Fail fast with a clear pointer
	// to `dagobert update` when it is missing entirely.
	guardMitre("mitre/enterprise-attack.json", "mitre/ics-attack.json", "mitre/mobile-attack.json")
	mitre, err := attck.LoadKB(
		"mitre/enterprise-attack.json",
		"mitre/ics-attack.json",
		"mitre/mobile-attack.json",
	)
	if err != nil {
		slog.Error("Failed to load MITRE ATT&CK knowledge base", "err", err)
		return
	}

	// --------------------------------------
	// Timesketch
	// --------------------------------------
	// shared lazy client; an empty TIMESKETCH_URL yields an unconfigured
	// client that fails with a friendly error on use
	ts := timesketch.NewClient(timesketch.Config{
		URL:           os.Getenv("TIMESKETCH_URL"),
		Username:      os.Getenv("TIMESKETCH_USER"),
		Password:      os.Getenv("TIMESKETCH_PASS"),
		SkipVerifyTLS: os.Getenv("TIMESKETCH_SKIP_VERIFY_TLS") == "true",
	})

	// --------------------------------------
	// Automations & jobs
	// --------------------------------------
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go modules.Start(ctx, db, ts)
	defer stop()

	// --------------------------------------
	// Router
	// --------------------------------------
	slog.Debug("Creating router and registering handlers")
	router := http.NewServeMux()
	secured := http.NewServeMux()
	securedH := a.Require(acl.Protect(secured))
	a.SetRoutes(secured)

	// index
	secured.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cases/", http.StatusTemporaryRedirect)
	})

	// change password (requires a logged-in user)
	secured.HandleFunc("GET /auth/changepassword", a.ChangePasswordForm)
	secured.HandleFunc("POST /auth/changepassword", a.ChangePassword)

	// cases
	h := &Handler{Store: db, ACL: acl, Mitre: mitre, Timesketch: ts}
	secured.HandleFunc("GET /cases/", h.CaseList)
	secured.HandleFunc("GET /cases/export/csv", h.CaseExport)
	secured.HandleFunc("GET /cases/import/csv", h.CaseImport)
	secured.HandleFunc("POST /cases/import/csv", h.CaseImport)
	secured.HandleFunc("GET /cases/switch", h.CaseSwitch)
	secured.HandleFunc("GET /cases/import/archive", h.ImportArchiveForm)
	secured.HandleFunc("POST /cases/import/archive", h.ImportArchive)
	secured.HandleFunc("GET /cases/{cid}/export/archive", h.ExportArchive)
	secured.HandleFunc("GET /cases/{cid}", h.CaseEdit)
	secured.HandleFunc("POST /cases/{cid}", h.CaseSave)
	secured.HandleFunc("DELETE /cases/{cid}", h.CaseDelete)
	secured.HandleFunc("GET /cases/{cid}/fork", h.CaseForkEdit)
	secured.HandleFunc("POST /cases/{cid}/fork", h.CaseForkSave)
	secured.HandleFunc("GET /cases/{cid}/acl", h.CaseEditACL)
	secured.HandleFunc("POST /cases/{cid}/acl", h.CaseSaveACL)
	secured.HandleFunc("GET /cases/{cid}/summary/", h.CaseSummary)

	// users
	secured.HandleFunc("GET /settings/users/", h.UserList)
	secured.HandleFunc("GET /settings/users/{id}", h.UserEdit)
	secured.HandleFunc("POST /settings/users/{id}", h.UserSave)
	secured.HandleFunc("DELETE /settings/users/{id}", h.UserDelete)
	secured.HandleFunc("GET /settings/users/{id}/acl", h.UserEditACL)
	secured.HandleFunc("POST /settings/users/{id}/acl", h.UserSaveACL)

	// api keys
	secured.HandleFunc("GET /settings/api-keys/", h.KeyList)
	secured.HandleFunc("GET /settings/api-keys/{key}", h.KeyEdit)
	secured.HandleFunc("POST /settings/api-keys/{key}", h.KeySave)
	secured.HandleFunc("DELETE /settings/api-keys/{key}", h.KeyDelete)

	// settings (report templates)
	secured.HandleFunc("GET /settings/", h.Overview)
	secured.HandleFunc("GET /settings/reports/", h.ListReports)
	secured.HandleFunc("GET /settings/reports/{id}", h.EditReport)
	secured.HandleFunc("POST /settings/reports/{id}", h.SaveReport)
	secured.HandleFunc("DELETE /settings/reports/{id}", h.DeleteReport)

	// settings (case templates)
	secured.HandleFunc("GET /settings/templates/", h.ListTemplates)
	secured.HandleFunc("GET /settings/templates/promote", h.PromoteForm)
	secured.HandleFunc("POST /settings/templates/promote", h.Promote)
	secured.HandleFunc("GET /settings/templates/{cid}", h.EditTemplate)
	secured.HandleFunc("POST /settings/templates/{cid}", h.SaveTemplate)
	secured.HandleFunc("DELETE /settings/templates/{cid}", h.DeleteTemplate)

	// settings (hooks)
	secured.HandleFunc("GET /settings/hooks/", h.ListHooks)
	secured.HandleFunc("GET /settings/hooks/{id}", h.EditHook)
	secured.HandleFunc("POST /settings/hooks/{id}", h.SaveHook)
	secured.HandleFunc("DELETE /settings/hooks/{id}", h.DeleteHook)

	// settings (enums)
	secured.HandleFunc("GET /settings/enums/", h.ListEnums)
	secured.HandleFunc("GET /settings/enums/{id}", h.EditEnum)
	secured.HandleFunc("POST /settings/enums/{id}", h.SaveEnum)
	secured.HandleFunc("DELETE /settings/enums/{id}", h.DeleteEnum)

	// settings (custom attributes)
	secured.HandleFunc("GET /settings/custom/", h.ListCustomAttributes)
	secured.HandleFunc("GET /settings/custom/{id}", h.EditCustomAttribute)
	secured.HandleFunc("POST /settings/custom/{id}", h.SaveCustomAttribute)
	secured.HandleFunc("DELETE /settings/custom/{id}", h.DeleteCustomAttribute)

	// events
	secured.HandleFunc("GET /cases/{cid}/events/", h.EventList)
	secured.HandleFunc("GET /cases/{cid}/events/export/csv", h.EventExport)
	secured.HandleFunc("GET /cases/{cid}/events/import/csv", h.EventImportCSV)
	secured.HandleFunc("POST /cases/{cid}/events/import/csv", h.EventImportCSV)
	secured.HandleFunc("POST /cases/{cid}/events/import/timesketch", h.EventImportTimesketch)
	secured.HandleFunc("GET /cases/{cid}/events/{id}", h.EventEdit)
	secured.HandleFunc("POST /cases/{cid}/events/{id}", h.EventSave)
	secured.HandleFunc("DELETE /cases/{cid}/events/{id}", h.EventDelete)

	// assets
	secured.HandleFunc("GET /cases/{cid}/assets/", h.AssetList)
	secured.HandleFunc("GET /cases/{cid}/assets/export/csv", h.AssetExport)
	secured.HandleFunc("GET /cases/{cid}/assets/import/csv", h.AssetImport)
	secured.HandleFunc("POST /cases/{cid}/assets/import/csv", h.AssetImport)
	secured.HandleFunc("GET /cases/{cid}/assets/{id}", h.AssetEdit)
	secured.HandleFunc("POST /cases/{cid}/assets/{id}", h.AssetSave)
	secured.HandleFunc("DELETE /cases/{cid}/assets/{id}", h.AssetDelete)

	// malware
	secured.HandleFunc("GET /cases/{cid}/malware/", h.MalwareList)
	secured.HandleFunc("GET /cases/{cid}/malware/export/csv", h.MalwareExport)
	secured.HandleFunc("GET /cases/{cid}/malware/import/csv", h.MalwareImport)
	secured.HandleFunc("POST /cases/{cid}/malware/import/csv", h.MalwareImport)
	secured.HandleFunc("GET /cases/{cid}/malware/{id}", h.MalwareEdit)
	secured.HandleFunc("GET /cases/{cid}/malware/{id}/download", h.MalwareDownload)
	secured.HandleFunc("POST /cases/{cid}/malware/{id}", h.MalwareSave)
	secured.HandleFunc("DELETE /cases/{cid}/malware/{id}", h.MalwareDelete)
	secured.HandleFunc("GET /cases/{cid}/malware/{id}/run", h.MalwareListModules)
	secured.HandleFunc("POST /cases/{cid}/malware/{id}/run", h.MalwareScheduleModule)

	// indicators
	secured.HandleFunc("GET /cases/{cid}/indicators/", h.IndicatorList)
	secured.HandleFunc("GET /cases/{cid}/indicators/export/csv", h.IndicatorExportCSV)
	secured.HandleFunc("GET /cases/{cid}/indicators/export/ioc", h.IndicatorExportOpenIOC)
	secured.HandleFunc("GET /cases/{cid}/indicators/export/stix", h.IndicatorExportStix)
	secured.HandleFunc("GET /cases/{cid}/indicators/import/csv", h.IndicatorImportCSV)
	secured.HandleFunc("POST /cases/{cid}/indicators/import/csv", h.IndicatorImportCSV)
	secured.HandleFunc("POST /cases/{cid}/indicators/import/timesketch", h.IndicatorImportTimesketch)
	secured.HandleFunc("GET /cases/{cid}/indicators/{id}", h.IndicatorEdit)
	secured.HandleFunc("POST /cases/{cid}/indicators/{id}", h.IndicatorSave)
	secured.HandleFunc("DELETE /cases/{cid}/indicators/{id}", h.IndicatorDelete)
	secured.HandleFunc("GET /cases/{cid}/indicators/{id}/run", h.IndicatorListModules)
	secured.HandleFunc("POST /cases/{cid}/indicators/{id}/run", h.IndicatorScheduleModule)

	// evidence
	secured.HandleFunc("GET /cases/{cid}/evidences/", h.EvidenceList)
	secured.HandleFunc("GET /cases/{cid}/evidences/export/csv", h.EvidenceExport)
	secured.HandleFunc("GET /cases/{cid}/evidences/import/csv", h.EvidenceImport)
	secured.HandleFunc("POST /cases/{cid}/evidences/import/csv", h.EvidenceImport)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}", h.EvidenceEdit)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}/download", h.EvidenceDownload)
	secured.HandleFunc("POST /cases/{cid}/evidences/{id}", h.EvidenceSave)
	secured.HandleFunc("DELETE /cases/{cid}/evidences/{id}", h.EvidenceDelete)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}/run", h.EvidenceListModules)
	secured.HandleFunc("POST /cases/{cid}/evidences/{id}/run", h.EvidenceScheduleModule)

	// tasks
	secured.HandleFunc("GET /cases/{cid}/tasks/", h.TaskList)
	secured.HandleFunc("GET /cases/{cid}/tasks/export/csv", h.TaskExport)
	secured.HandleFunc("GET /cases/{cid}/tasks/import/csv", h.TaskImport)
	secured.HandleFunc("POST /cases/{cid}/tasks/import/csv", h.TaskImport)
	secured.HandleFunc("GET /cases/{cid}/tasks/{id}", h.TaskEdit)
	secured.HandleFunc("POST /cases/{cid}/tasks/{id}", h.TaskSave)
	secured.HandleFunc("DELETE /cases/{cid}/tasks/{id}", h.TaskDelete)

	// notes
	secured.HandleFunc("GET /cases/{cid}/notes/", h.NoteList)
	secured.HandleFunc("GET /cases/{cid}/notes/export/csv", h.NoteExport)
	secured.HandleFunc("GET /cases/{cid}/notes/import/csv", h.NoteImport)
	secured.HandleFunc("POST /cases/{cid}/notes/import/csv", h.NoteImport)
	secured.HandleFunc("GET /cases/{cid}/notes/{id}", h.NoteEdit)
	secured.HandleFunc("POST /cases/{cid}/notes/{id}", h.NoteSave)
	secured.HandleFunc("DELETE /cases/{cid}/notes/{id}", h.NoteDelete)

	// comments (on case sub-objects)
	secured.HandleFunc("GET /cases/{cid}/comments/{kind}/{oid}/", h.CommentList)
	secured.HandleFunc("POST /cases/{cid}/comments/{kind}/{oid}/{id}", h.CommentSave)
	secured.HandleFunc("DELETE /cases/{cid}/comments/{kind}/{oid}/{id}", h.CommentDelete)

	// visualizations
	secured.HandleFunc("GET /cases/{cid}/vis/network", h.Network)
	secured.HandleFunc("GET /cases/{cid}/vis/mitre", h.MitreAttack)

	// reports
	secured.HandleFunc("GET /cases/{cid}/reports", h.ReportDialog)
	secured.HandleFunc("POST /cases/{cid}/render", h.ReportGenerate)

	// read-only MCP server (stateless Streamable-HTTP)
	secured.Handle("/mcp", NewMcpHandler(db))

	// test routes
	router.HandleFunc("GET /errors/400", Serve4xx)
	router.HandleFunc("GET /errors/500", Serve5xx)

	// static assets
	router.Handle("GET /public/", ServeDir("/public/", public.AssetsFS))

	// auth routes (unauthenticated)
	router.HandleFunc("GET /auth/login", a.LoginLocal)
	router.HandleFunc("POST /auth/login", a.LoginLocal)
	router.HandleFunc("GET /auth/oidc", a.LoginOIDC)
	router.HandleFunc("GET /auth/callback", a.Callback)
	router.HandleFunc("GET /auth/logout", a.Logout)
	router.Handle("/", securedH)

	// --------------------------------------
	// Server
	// --------------------------------------
	// Order matters: Session.LoadAndSave must wrap LoadUser (which reads the
	// session), and ApiKeyMiddleware wraps both so its Cookie/Authorization
	// strip happens before any session state is loaded and the system user it
	// sets is found by LoadUser's context check.
	var chain http.Handler = router
	chain = a.LoadUser(chain)
	chain = Session.LoadAndSave(chain)
	chain = auth.ApiKeyMiddleware(db)(chain) // strips browser credentials before session state is loaded
	chain = http.NewCrossOriginProtection().Handler(chain)
	chain = Logger(chain)
	chain = SecurityHeaders(chain)
	chain = Recover(chain) // outermost: a panic anywhere (incl. the layers below) must not take the server down

	srv := &http.Server{Addr: ":8080", Handler: chain}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()

	slog.Info("Starting web server", "addr", ":8080")
	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start web server", "err", err)
		return
	}
}

// Session is the application-wide Session manager. It is initialized by
// initSession during handler.Run and shared with the auth layer.
var Session *scs.SessionManager

func initSession(db *sql.DB) {
	Session = scs.New()
	Session.Store = sqlite3store.New(db) // runs its own expired-session cleanup goroutine
	Session.Lifetime = 24 * time.Hour
	Session.Cookie.HttpOnly = true
	Session.Cookie.SameSite = http.SameSiteLaxMode
	// HTTPS-only by default; relax for local development over plain HTTP.
	Session.Cookie.Secure = os.Getenv("WEB_SECURE") != "false"
}

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
