package main

import (
	"cmp"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lmittmann/tint"
	"github.com/sprungknoedl/dagobert/app/handler"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/public"
)

type Configuration struct {
	AssetsFolder   string
	EvidenceFolder string

	Database string

	ClientId      string
	ClientSecret  string
	ClientUrl     string
	Issuer        string
	IdentityClaim string

	SessionSecret string
}

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.DateTime,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// write all errors in red
				if err, ok := a.Value.Any().(error); ok {
					aErr := tint.Err(err)
					aErr.Key = a.Key
					return aErr
				}
				return a
			},
		}),
	))

	if len(os.Args) > 1 && os.Args[1] == "worker" {
		worker.StartWorker()
	} else {
		StartUI()
	}
}

func StartUI() {
	cfg := Configuration{
		AssetsFolder:   cmp.Or(os.Getenv("FS_ASSETS_FOLDER"), "./web"),
		EvidenceFolder: cmp.Or(os.Getenv("FS_EVIDENCE_FOLDER"), "./files/evidences"),
		Database:       cmp.Or(os.Getenv("DB_URL"), "file:files/dagobert.db?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)"),
		ClientId:       os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret:   os.Getenv("OIDC_CLIENT_SECRET"),
		ClientUrl:      os.Getenv("OIDC_CLIENT_URL"),
		Issuer:         os.Getenv("OIDC_ISSUER"),
		IdentityClaim:  cmp.Or(os.Getenv("OIDC_ID_CLAIM"), "sub"),
		SessionSecret:  os.Getenv("WEB_SESSION_SECRET"),
	}

	slog.Debug("Connecting to database", "url", cfg.Database)
	db, err := model.Connect(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database: %v", "err", err)
		return
	}

	slog.Debug("Initializing database")
	err = InitializeDatabase(db)
	if err != nil {
		slog.Error("Failed to run database migrations: %v", "err", err)
		return
	}

	slog.Debug("Creating timesketch client", "url", os.Getenv("TIMESKETCH_URL"))
	ts, err := timesketch.NewClient(
		os.Getenv("TIMESKETCH_URL"),
		os.Getenv("TIMESKETCH_USER"),
		os.Getenv("TIMESKETCH_PASS"),
	)
	if err != nil {
		slog.Warn("Failed to create timesketch client: %v", "err", err)
	}

	// --------------------------------------
	// Authorization
	// --------------------------------------
	slog.Debug("Creating casbin acl model")
	acl := handler.NewACL(db)

	// --------------------------------------
	// Authentication
	// --------------------------------------
	slog.Debug("Creating oid provider", "url", cfg.Issuer)
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	auth := handler.NewAuthCtrl(db, acl, handler.OpenIDConfig{
		ClientId:      cfg.ClientId,
		ClientSecret:  cfg.ClientSecret,
		Issuer:        *issuer,
		ClientUrl:     *clientUrl,
		Identifier:    cfg.IdentityClaim,
		AutoProvision: os.Getenv("OIDC_AUTO_PROVISION") == "true",
		Scopes:        []string{"openid", "profile", "email"},
		PostLogoutUrl: *clientUrl,
	})

	// --------------------------------------
	// Router
	// --------------------------------------
	slog.Debug("Creating router and registering handlers")
	mux := http.NewServeMux()
	srv := handler.Recover(mux)
	srv = auth.Protect(srv)
	srv = handler.Logger(srv)

	// --------------------------------------
	// Home
	// --------------------------------------
	// index
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cases/", http.StatusTemporaryRedirect)
	})

	// auth
	mux.HandleFunc("GET /auth/logout", auth.Logout)
	mux.HandleFunc("GET /auth/callback", auth.Callback)
	mux.HandleFunc("GET /auth/forbidden", auth.Forbidden)

	// cases
	caseCtrl := handler.NewCaseCtrl(db, acl, ts)
	mux.HandleFunc("GET /cases/", caseCtrl.List)
	mux.HandleFunc("GET /cases/export/csv", caseCtrl.Export)
	mux.HandleFunc("GET /cases/import/csv", caseCtrl.Import)
	mux.HandleFunc("POST /cases/import/csv", caseCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}", caseCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}", caseCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}", caseCtrl.Delete)
	mux.HandleFunc("GET /settings/cases/{cid}/acl", caseCtrl.EditACL)
	mux.HandleFunc("POST /settings/cases/{cid}/acl", caseCtrl.SaveACL)
	mux.HandleFunc("GET /cases/{cid}/summary/", caseCtrl.Summary)

	// users
	userCtrl := handler.NewUserCtrl(db, acl)
	mux.HandleFunc("GET /settings/users/", userCtrl.List)
	mux.HandleFunc("GET /settings/users/{id}", userCtrl.Edit)
	mux.HandleFunc("POST /settings/users/{id}", userCtrl.Save)
	mux.HandleFunc("DELETE /settings/users/{id}", userCtrl.Delete)
	mux.HandleFunc("GET /settings/users/{id}/acl", userCtrl.EditACL)
	mux.HandleFunc("POST /settings/users/{id}/acl", userCtrl.SaveACL)

	// evidence processing jobs
	jobCtrl := handler.NewJobCtrl(db, acl)
	mux.HandleFunc("GET /internal/jobs", jobCtrl.PopJob)
	mux.HandleFunc("POST /internal/jobs/ack", jobCtrl.AckJob)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}/run", jobCtrl.ListMods)
	mux.HandleFunc("POST /cases/{cid}/evidences/{id}/run", jobCtrl.PushJob)

	// api keys
	keyCtrl := handler.NewKeyCtrl(db, acl, jobCtrl)
	mux.HandleFunc("GET /settings/api-keys/", keyCtrl.List)
	mux.HandleFunc("GET /settings/api-keys/{key}", keyCtrl.Edit)
	mux.HandleFunc("POST /settings/api-keys/{key}", keyCtrl.Save)
	mux.HandleFunc("DELETE /settings/api-keys/{key}", keyCtrl.Delete)

	// settings (report templates)
	settingsCtrl := handler.NewSettingsCtrl(db, acl)
	mux.HandleFunc("GET /settings/reports/", settingsCtrl.ListReports)
	mux.HandleFunc("GET /settings/reports/{id}", settingsCtrl.EditReport)
	mux.HandleFunc("POST /settings/reports/{id}", settingsCtrl.SaveReport)
	mux.HandleFunc("DELETE /settings/reports/{id}", settingsCtrl.DeleteReport)

	// settings (hooks)
	mux.HandleFunc("GET /settings/hooks/", settingsCtrl.ListHooks)
	mux.HandleFunc("GET /settings/hooks/{id}", settingsCtrl.EditHook)
	mux.HandleFunc("POST /settings/hooks/{id}", settingsCtrl.SaveHook)
	mux.HandleFunc("DELETE /settings/hooks/{id}", settingsCtrl.DeleteHook)

	// settings (enums)
	mux.HandleFunc("GET /settings/enums/", settingsCtrl.ListEnums)
	mux.HandleFunc("GET /settings/enums/{id}", settingsCtrl.EditEnum)
	mux.HandleFunc("POST /settings/enums/{id}", settingsCtrl.SaveEnum)
	mux.HandleFunc("DELETE /settings/enums/{id}", settingsCtrl.DeleteEnum)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	eventCtrl := handler.NewEventCtrl(db, acl, ts)
	mux.HandleFunc("GET /cases/{cid}/events/", eventCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/events/export/csv", eventCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/events/import/csv", eventCtrl.ImportCSV)
	mux.HandleFunc("POST /cases/{cid}/events/import/csv", eventCtrl.ImportCSV)
	mux.HandleFunc("POST /cases/{cid}/events/import/timesketch", eventCtrl.ImportTimesketch)
	mux.HandleFunc("GET /cases/{cid}/events/{id}", eventCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/events/{id}", eventCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/events/{id}", eventCtrl.Delete)

	// assets
	assetCtrl := handler.NewAssetCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/assets/", assetCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/assets/export/csv", assetCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/assets/import/csv", assetCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/assets/import/csv", assetCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/assets/{id}", assetCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/assets/{id}", assetCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/assets/{id}", assetCtrl.Delete)

	// malware
	malwareCtrl := handler.NewMalwareCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/malware/", malwareCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/malware/export/csv", malwareCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/malware/import/csv", malwareCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/malware/import/csv", malwareCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/malware/{id}", malwareCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/malware/{id}", malwareCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/malware/{id}", malwareCtrl.Delete)

	// indicators
	indicatorCtrl := handler.NewIndicatorCtrl(db, acl, ts)
	mux.HandleFunc("GET /cases/{cid}/indicators/", indicatorCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/indicators/export/csv", indicatorCtrl.ExportCSV)
	mux.HandleFunc("GET /cases/{cid}/indicators/export/ioc", indicatorCtrl.ExportOpenIOC)
	mux.HandleFunc("GET /cases/{cid}/indicators/export/stix", indicatorCtrl.ExportStix)
	mux.HandleFunc("GET /cases/{cid}/indicators/import/csv", indicatorCtrl.ImportCSV)
	mux.HandleFunc("POST /cases/{cid}/indicators/import/csv", indicatorCtrl.ImportCSV)
	mux.HandleFunc("POST /cases/{cid}/indicators/import/timesketch", indicatorCtrl.ImportTimesketch)
	mux.HandleFunc("GET /cases/{cid}/indicators/{id}", indicatorCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/indicators/{id}", indicatorCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/indicators/{id}", indicatorCtrl.Delete)

	// evidence
	evidenceCtrl := handler.NewEvidenceCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/evidences/", evidenceCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/evidences/export/csv", evidenceCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/evidences/import/csv", evidenceCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/evidences/import/csv", evidenceCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}", evidenceCtrl.Edit)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}/download", evidenceCtrl.Download)
	mux.HandleFunc("POST /cases/{cid}/evidences/{id}", evidenceCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/evidences/{id}", evidenceCtrl.Delete)

	// tasks
	taskCtrl := handler.NewTaskCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/tasks/", taskCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/tasks/export/csv", taskCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/tasks/import/csv", taskCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/tasks/import/csv", taskCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/tasks/{id}", taskCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/tasks/{id}", taskCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/tasks/{id}", taskCtrl.Delete)

	// notes
	noteCtrl := handler.NewNoteCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/notes/", noteCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/notes/export/csv", noteCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/notes/import/csv", noteCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/notes/import/csv", noteCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/notes/{id}", noteCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/notes/{id}", noteCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/notes/{id}", noteCtrl.Delete)

	// visualizations
	visualsCtrl := handler.NewVisualsCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/vis/network", visualsCtrl.Network)
	mux.HandleFunc("GET /cases/{cid}/vis/timeline", visualsCtrl.Timeline)

	// reports
	reportsCtrl := handler.NewReportsCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/reports", reportsCtrl.Dialog)
	mux.HandleFunc("POST /cases/{cid}/render", reportsCtrl.Generate)

	// test routes
	mux.HandleFunc("GET /errors/400", handler.Serve4xx)
	mux.HandleFunc("GET /errors/500", handler.Serve5xx)

	// --------------------------------------
	// Static Assets
	// --------------------------------------
	mux.Handle("GET /favicon.ico", handler.ServeFile(filepath.Join(cfg.AssetsFolder, "favicon.ico")))
	mux.Handle("GET /public/", handler.ServeDir("/public/", public.AssetsFS))

	// --------------------------------------
	// Initialize Dagobert
	// --------------------------------------
	slog.Debug("Initializing dagobert")
	err = InitializeDagobert(db, acl, cfg)
	if err != nil {
		slog.Error("Failed to initialize dagobert", "err", err)
		return
	}

	slog.Debug("Loading hooks")
	err = handler.LoadHooks(db)
	if err != nil {
		slog.Error("Failed to load hooks", "err", err)
		return
	}

	slog.Debug("Rescheduling stale jobs")
	err = db.RescheduleStaleJobs()
	if err != nil {
		slog.Error("Failed to reschedule state jobs", "err", err)
	}

	// TODO: make lsiten address configurable
	slog.Info("Starting web server", "addr", ":8080")
	err = http.ListenAndServe(":8080", srv)
	if err != nil {
		slog.Error("Failed to start web server", "err", err)
		return
	}
}

func InitializeDatabase(store *model.Store) error {
	db, err := sqlite.WithInstance(store.RawConn, &sqlite.Config{})
	if err != nil {
		return err
	}

	slog.Debug("Loading database migrations")
	source, err := iofs.New(model.Migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", db)
	if err != nil {
		return err
	}

	slog.Debug("Applying database migrations")
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

func InitializeDagobert(store *model.Store, acl *handler.ACL, cfg Configuration) error {
	users, err := store.ListUsers()
	if err != nil {
		return err
	}

	slog.Debug("Initializing dagobert, found users", "count", len(users))
	if len(users) == 0 {
		// initialize administrators
		slog.Info("Initializing administrators")
		for _, env := range os.Environ() {
			if !strings.HasPrefix(env, "DAGOBERT_ADMIN_") {
				continue
			}

			key, value, _ := strings.Cut(env, "=")
			slog.Info("Adding administrator", "uid", value)
			err = store.SaveUser(model.User{
				ID:   value,
				UPN:  key,
				Name: key,
				Role: "Administrator",
			})
			if err != nil {
				return err
			}

			err = acl.SaveUserRole(value, "Administrator")
			if err != nil {
				return err
			}
		}
	}

	keys, err := store.ListKeys()
	if err != nil {
		return err
	}

	slog.Debug("Initializing dagobert, found key", "count", len(keys))
	if len(keys) == 0 {
		// initialize api keys
		slog.Info("Initializing api keys")
		for _, env := range os.Environ() {
			if !strings.HasPrefix(env, "DAGOBERT_KEY_") {
				continue
			}

			key, value, _ := strings.Cut(env, "=")
			slog.Info("Adding api key", "key", value)
			err = store.SaveKey(model.Key{
				Type: "Dagobert",
				Key:  value,
				Name: key,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
