package handler

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker"
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
	InitSession(db.RawConn)
	a, err := auth.New(db, Session)
	if err != nil {
		slog.Error("Failed to initialize auth", "err", err)
		return
	}

	// --------------------------------------
	// MITRE ATT&CK
	// --------------------------------------
	// system data lives outside files/, which holds user data only (and is
	// shadowed by the volume mount in Docker)
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
	go worker.Start(ctx, db, ts)
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
	caseCtrl := NewCaseCtrl(db, acl, ts)
	secured.HandleFunc("GET /cases/", caseCtrl.List)
	secured.HandleFunc("GET /cases/export/csv", caseCtrl.Export)
	secured.HandleFunc("GET /cases/import/csv", caseCtrl.Import)
	secured.HandleFunc("POST /cases/import/csv", caseCtrl.Import)
	secured.HandleFunc("GET /cases/switch", caseCtrl.Switch)
	secured.HandleFunc("GET /cases/import/archive", caseCtrl.ImportArchiveForm)
	secured.HandleFunc("POST /cases/import/archive", caseCtrl.ImportArchive)
	secured.HandleFunc("GET /cases/{cid}/export/archive", caseCtrl.ExportArchive)
	secured.HandleFunc("GET /cases/{cid}", caseCtrl.Edit)
	secured.HandleFunc("POST /cases/{cid}", caseCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}", caseCtrl.Delete)
	secured.HandleFunc("GET /cases/{cid}/acl", caseCtrl.EditACL)
	secured.HandleFunc("POST /cases/{cid}/acl", caseCtrl.SaveACL)
	secured.HandleFunc("GET /cases/{cid}/summary/", caseCtrl.Summary)

	// users
	userCtrl := NewUserCtrl(db, acl)
	secured.HandleFunc("GET /settings/users/", userCtrl.List)
	secured.HandleFunc("GET /settings/users/{id}", userCtrl.Edit)
	secured.HandleFunc("POST /settings/users/{id}", userCtrl.Save)
	secured.HandleFunc("DELETE /settings/users/{id}", userCtrl.Delete)
	secured.HandleFunc("GET /settings/users/{id}/acl", userCtrl.EditACL)
	secured.HandleFunc("POST /settings/users/{id}/acl", userCtrl.SaveACL)

	// api keys
	keyCtrl := NewKeyCtrl(db, acl)
	secured.HandleFunc("GET /settings/api-keys/", keyCtrl.List)
	secured.HandleFunc("GET /settings/api-keys/{key}", keyCtrl.Edit)
	secured.HandleFunc("POST /settings/api-keys/{key}", keyCtrl.Save)
	secured.HandleFunc("DELETE /settings/api-keys/{key}", keyCtrl.Delete)

	// settings (report templates)
	settingsCtrl := NewSettingsCtrl(db, acl)
	secured.HandleFunc("GET /settings/", settingsCtrl.Overview)
	secured.HandleFunc("GET /settings/reports/", settingsCtrl.ListReports)
	secured.HandleFunc("GET /settings/reports/{id}", settingsCtrl.EditReport)
	secured.HandleFunc("POST /settings/reports/{id}", settingsCtrl.SaveReport)
	secured.HandleFunc("DELETE /settings/reports/{id}", settingsCtrl.DeleteReport)

	// settings (case templates)
	secured.HandleFunc("GET /settings/templates/", settingsCtrl.ListTemplates)
	secured.HandleFunc("GET /settings/templates/promote", settingsCtrl.PromoteForm)
	secured.HandleFunc("POST /settings/templates/promote", settingsCtrl.Promote)
	secured.HandleFunc("GET /settings/templates/{cid}", settingsCtrl.EditTemplate)
	secured.HandleFunc("POST /settings/templates/{cid}", settingsCtrl.SaveTemplate)
	secured.HandleFunc("DELETE /settings/templates/{cid}", settingsCtrl.DeleteTemplate)

	// settings (hooks)
	secured.HandleFunc("GET /settings/hooks/", settingsCtrl.ListHooks)
	secured.HandleFunc("GET /settings/hooks/{id}", settingsCtrl.EditHook)
	secured.HandleFunc("POST /settings/hooks/{id}", settingsCtrl.SaveHook)
	secured.HandleFunc("DELETE /settings/hooks/{id}", settingsCtrl.DeleteHook)

	// settings (enums)
	secured.HandleFunc("GET /settings/enums/", settingsCtrl.ListEnums)
	secured.HandleFunc("GET /settings/enums/{id}", settingsCtrl.EditEnum)
	secured.HandleFunc("POST /settings/enums/{id}", settingsCtrl.SaveEnum)
	secured.HandleFunc("DELETE /settings/enums/{id}", settingsCtrl.DeleteEnum)

	// settings (custom attributes)
	secured.HandleFunc("GET /settings/custom/", settingsCtrl.ListCustomAttributes)
	secured.HandleFunc("GET /settings/custom/{id}", settingsCtrl.EditCustomAttribute)
	secured.HandleFunc("POST /settings/custom/{id}", settingsCtrl.SaveCustomAttribute)
	secured.HandleFunc("DELETE /settings/custom/{id}", settingsCtrl.DeleteCustomAttribute)

	// events
	eventCtrl := NewEventCtrl(db, acl, mitre, ts)
	secured.HandleFunc("GET /cases/{cid}/events/", eventCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/events/export/csv", eventCtrl.Export)
	secured.HandleFunc("GET /cases/{cid}/events/import/csv", eventCtrl.ImportCSV)
	secured.HandleFunc("POST /cases/{cid}/events/import/csv", eventCtrl.ImportCSV)
	secured.HandleFunc("POST /cases/{cid}/events/import/timesketch", eventCtrl.ImportTimesketch)
	secured.HandleFunc("GET /cases/{cid}/events/{id}", eventCtrl.Edit)
	secured.HandleFunc("POST /cases/{cid}/events/{id}", eventCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/events/{id}", eventCtrl.Delete)

	// assets
	assetCtrl := NewAssetCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/assets/", assetCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/assets/export/csv", assetCtrl.Export)
	secured.HandleFunc("GET /cases/{cid}/assets/import/csv", assetCtrl.Import)
	secured.HandleFunc("POST /cases/{cid}/assets/import/csv", assetCtrl.Import)
	secured.HandleFunc("GET /cases/{cid}/assets/{id}", assetCtrl.Edit)
	secured.HandleFunc("POST /cases/{cid}/assets/{id}", assetCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/assets/{id}", assetCtrl.Delete)

	// malware
	malwareCtrl := NewMalwareCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/malware/", malwareCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/malware/export/csv", malwareCtrl.Export)
	secured.HandleFunc("GET /cases/{cid}/malware/import/csv", malwareCtrl.Import)
	secured.HandleFunc("POST /cases/{cid}/malware/import/csv", malwareCtrl.Import)
	secured.HandleFunc("GET /cases/{cid}/malware/{id}", malwareCtrl.Edit)
	secured.HandleFunc("GET /cases/{cid}/malware/{id}/download", malwareCtrl.Download)
	secured.HandleFunc("POST /cases/{cid}/malware/{id}", malwareCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/malware/{id}", malwareCtrl.Delete)
	secured.HandleFunc("GET /cases/{cid}/malware/{id}/run", malwareCtrl.ListModules)
	secured.HandleFunc("POST /cases/{cid}/malware/{id}/run", malwareCtrl.ScheduleModule)

	// indicators
	indicatorCtrl := NewIndicatorCtrl(db, acl, ts)
	secured.HandleFunc("GET /cases/{cid}/indicators/", indicatorCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/indicators/export/csv", indicatorCtrl.ExportCSV)
	secured.HandleFunc("GET /cases/{cid}/indicators/export/ioc", indicatorCtrl.ExportOpenIOC)
	secured.HandleFunc("GET /cases/{cid}/indicators/export/stix", indicatorCtrl.ExportStix)
	secured.HandleFunc("GET /cases/{cid}/indicators/import/csv", indicatorCtrl.ImportCSV)
	secured.HandleFunc("POST /cases/{cid}/indicators/import/csv", indicatorCtrl.ImportCSV)
	secured.HandleFunc("POST /cases/{cid}/indicators/import/timesketch", indicatorCtrl.ImportTimesketch)
	secured.HandleFunc("GET /cases/{cid}/indicators/{id}", indicatorCtrl.Edit)
	secured.HandleFunc("POST /cases/{cid}/indicators/{id}", indicatorCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/indicators/{id}", indicatorCtrl.Delete)
	secured.HandleFunc("GET /cases/{cid}/indicators/{id}/run", indicatorCtrl.ListModules)
	secured.HandleFunc("POST /cases/{cid}/indicators/{id}/run", indicatorCtrl.ScheduleModule)

	// evidence
	evidenceCtrl := NewEvidenceCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/evidences/", evidenceCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/evidences/export/csv", evidenceCtrl.Export)
	secured.HandleFunc("GET /cases/{cid}/evidences/import/csv", evidenceCtrl.Import)
	secured.HandleFunc("POST /cases/{cid}/evidences/import/csv", evidenceCtrl.Import)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}", evidenceCtrl.Edit)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}/download", evidenceCtrl.Download)
	secured.HandleFunc("POST /cases/{cid}/evidences/{id}", evidenceCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/evidences/{id}", evidenceCtrl.Delete)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}/run", evidenceCtrl.ListModules)
	secured.HandleFunc("POST /cases/{cid}/evidences/{id}/run", evidenceCtrl.ScheduleModule)

	// tasks
	taskCtrl := NewTaskCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/tasks/", taskCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/tasks/export/csv", taskCtrl.Export)
	secured.HandleFunc("GET /cases/{cid}/tasks/import/csv", taskCtrl.Import)
	secured.HandleFunc("POST /cases/{cid}/tasks/import/csv", taskCtrl.Import)
	secured.HandleFunc("GET /cases/{cid}/tasks/{id}", taskCtrl.Edit)
	secured.HandleFunc("POST /cases/{cid}/tasks/{id}", taskCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/tasks/{id}", taskCtrl.Delete)

	// notes
	noteCtrl := NewNoteCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/notes/", noteCtrl.List)
	secured.HandleFunc("GET /cases/{cid}/notes/export/csv", noteCtrl.Export)
	secured.HandleFunc("GET /cases/{cid}/notes/import/csv", noteCtrl.Import)
	secured.HandleFunc("POST /cases/{cid}/notes/import/csv", noteCtrl.Import)
	secured.HandleFunc("GET /cases/{cid}/notes/{id}", noteCtrl.Edit)
	secured.HandleFunc("POST /cases/{cid}/notes/{id}", noteCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/notes/{id}", noteCtrl.Delete)

	// visualizations
	visualsCtrl := NewVisualsCtrl(db, acl, mitre)
	secured.HandleFunc("GET /cases/{cid}/vis/network", visualsCtrl.Network)
	secured.HandleFunc("GET /cases/{cid}/vis/mitre", visualsCtrl.MitreAttack)

	// reports
	reportsCtrl := NewReportsCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/reports", reportsCtrl.Dialog)
	secured.HandleFunc("POST /cases/{cid}/render", reportsCtrl.Generate)

	// read-only MCP server (stateless Streamable-HTTP)
	mcpCtrl := NewMcpCtrl(db, acl)
	secured.Handle("/mcp", mcpCtrl)

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
	var h http.Handler = router
	h = a.LoadUser(h)
	h = Session.LoadAndSave(h)
	h = auth.ApiKeyMiddleware(db)(h) // strips browser credentials before session state is loaded
	h = http.NewCrossOriginProtection().Handler(h)
	h = Logger(h)
	h = SecurityHeaders(h)
	h = Recover(h) // outermost: a panic anywhere (incl. the layers below) must not take the server down

	srv := &http.Server{Addr: ":8080", Handler: h}
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
