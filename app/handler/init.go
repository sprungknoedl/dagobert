package handler

import (
	"cmp"
	"log/slog"
	"net/http"
	"os"

	"github.com/aarondl/authboss/v3"
	"github.com/justinas/alice"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/public"
)

func Run(cmd *cobra.Command, args []string) {
	// --------------------------------------
	// Database
	// --------------------------------------
	dburl := cmp.Or(os.Getenv("DB_URL"), "file:files/dagobert.db?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)")
	slog.Debug("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		slog.Error("Failed to connect to database", "err", err)
		return
	}

	// --------------------------------------
	// Authorization
	// --------------------------------------
	slog.Debug("Creating casbin acl model")
	acl := auth.NewACL(db)

	// --------------------------------------
	// Authentication
	// --------------------------------------
	ab, err := auth.Init(db)
	if err != nil {
		slog.Error("Failed to initialize authboss", "err", err)
		return
	}

	// --------------------------------------
	// Automations & jobs
	// --------------------------------------
	slog.Debug("Loading hooks")
	err = LoadHooks(db)
	if err != nil {
		slog.Error("Failed to load hooks", "err", err)
		return
	}

	slog.Debug("Rescheduling stale jobs")
	err = db.RescheduleStaleJobs()
	if err != nil {
		slog.Error("Failed to reschedule state jobs", "err", err)
	}

	// --------------------------------------
	// Router
	// --------------------------------------
	slog.Debug("Creating router and registering handlers")
	chain := alice.New(
		Recover,
		Logger,
		CSRF,
		ab.LoadClientStateMiddleware,
		auth.ApiKeyMiddleware(ab, db),
		authboss.ModuleListMiddleware(ab),
		acl.Protect)

	router := http.NewServeMux()
	secured := http.NewServeMux()
	securedH := authboss.Middleware2(ab, authboss.RequireNone, authboss.RespondRedirect)(secured)

	// index
	secured.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cases/", http.StatusTemporaryRedirect)
	})

	// cases
	caseCtrl := NewCaseCtrl(db, acl)
	secured.HandleFunc("GET /cases/", caseCtrl.List)
	secured.HandleFunc("GET /cases/export/csv", caseCtrl.Export)
	secured.HandleFunc("GET /cases/import/csv", caseCtrl.Import)
	secured.HandleFunc("POST /cases/import/csv", caseCtrl.Import)
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

	// evidence processing jobs
	jobCtrl := NewJobCtrl(db, acl)
	secured.HandleFunc("GET /internal/jobs", jobCtrl.PopJob)
	secured.HandleFunc("POST /internal/jobs/ack", jobCtrl.AckJob)
	secured.HandleFunc("GET /cases/{cid}/evidences/{id}/run", jobCtrl.ListMods)
	secured.HandleFunc("POST /cases/{cid}/evidences/{id}/run", jobCtrl.PushJob)

	// api keys
	keyCtrl := NewKeyCtrl(db, acl, jobCtrl)
	secured.HandleFunc("GET /settings/api-keys/", keyCtrl.List)
	secured.HandleFunc("GET /settings/api-keys/{key}", keyCtrl.Edit)
	secured.HandleFunc("POST /settings/api-keys/{key}", keyCtrl.Save)
	secured.HandleFunc("DELETE /settings/api-keys/{key}", keyCtrl.Delete)

	// settings (report templates)
	settingsCtrl := NewSettingsCtrl(db, acl)
	secured.HandleFunc("GET /settings/reports/", settingsCtrl.ListReports)
	secured.HandleFunc("GET /settings/reports/{id}", settingsCtrl.EditReport)
	secured.HandleFunc("POST /settings/reports/{id}", settingsCtrl.SaveReport)
	secured.HandleFunc("DELETE /settings/reports/{id}", settingsCtrl.DeleteReport)

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

	// events
	eventCtrl := NewEventCtrl(db, acl)
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
	secured.HandleFunc("POST /cases/{cid}/malware/{id}", malwareCtrl.Save)
	secured.HandleFunc("DELETE /cases/{cid}/malware/{id}", malwareCtrl.Delete)

	// indicators
	indicatorCtrl := NewIndicatorCtrl(db, acl)
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
	visualsCtrl := NewVisualsCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/vis/network", visualsCtrl.Network)
	secured.HandleFunc("GET /cases/{cid}/vis/timeline", visualsCtrl.Timeline)

	// reports
	reportsCtrl := NewReportsCtrl(db, acl)
	secured.HandleFunc("GET /cases/{cid}/reports", reportsCtrl.Dialog)
	secured.HandleFunc("POST /cases/{cid}/render", reportsCtrl.Generate)

	// test routes
	router.HandleFunc("GET /errors/400", Serve4xx)
	router.HandleFunc("GET /errors/500", Serve5xx)

	// static assets
	router.Handle("GET /public/", ServeDir("/public/", public.AssetsFS))

	// authboss
	router.Handle("/auth/", http.StripPrefix("/auth", ab.Config.Core.Router))
	router.Handle("/", securedH)

	// --------------------------------------
	// Server
	// --------------------------------------
	slog.Info("Starting web server", "addr", ":8080")
	err = http.ListenAndServe(":8080", chain.Then(router))
	if err != nil {
		slog.Error("Failed to start web server", "err", err)
		return
	}
}
