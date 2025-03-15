package main

import (
	"cmp"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/internal/handler"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/tty"
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

	db, err := model.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = InitializeDatabase(db)
	if err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	ts, err := timesketch.NewClient(
		os.Getenv("TIMESKETCH_URL"),
		os.Getenv("TIMESKETCH_USER"),
		os.Getenv("TIMESKETCH_PASS"),
	)
	if err != nil {
		log.Printf("Failed to create timesketch client: %v", err)
	}

	// --------------------------------------
	// Authorization
	// --------------------------------------
	acl := handler.NewACL(db)

	// --------------------------------------
	// Authentication
	// --------------------------------------
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
	mux.HandleFunc("GET /cases/export", caseCtrl.Export)
	mux.HandleFunc("GET /cases/import", caseCtrl.Import)
	mux.HandleFunc("POST /cases/import", caseCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}", caseCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}", caseCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}", caseCtrl.Delete)
	mux.HandleFunc("GET /settings/cases/{cid}/acl", caseCtrl.EditACL)
	mux.HandleFunc("POST /settings/cases/{cid}/acl", caseCtrl.SaveACL)

	// users
	userCtrl := handler.NewUserCtrl(db, acl)
	mux.HandleFunc("GET /settings/users/", userCtrl.List)
	mux.HandleFunc("GET /settings/users/{id}", userCtrl.Edit)
	mux.HandleFunc("POST /settings/users/{id}", userCtrl.Save)
	mux.HandleFunc("DELETE /settings/users/{id}", userCtrl.Delete)
	mux.HandleFunc("GET /settings/users/{id}/acl", userCtrl.EditACL)
	mux.HandleFunc("POST /settings/users/{id}/acl", userCtrl.SaveACL)

	// api keys
	keyCtrl := handler.NewKeyCtrl(db, acl)
	mux.HandleFunc("GET /settings/api-keys/", keyCtrl.List)
	mux.HandleFunc("GET /settings/api-keys/{key}", keyCtrl.Edit)
	mux.HandleFunc("POST /settings/api-keys/{key}", keyCtrl.Save)
	mux.HandleFunc("DELETE /settings/api-keys/{key}", keyCtrl.Delete)

	// settings (templates & hooks)
	settingsCtrl := handler.NewSettingsCtrl(db, acl)
	mux.HandleFunc("GET /settings/{$}", settingsCtrl.List)
	mux.HandleFunc("GET /settings/hooks/{id}", settingsCtrl.EditHook)
	mux.HandleFunc("POST /settings/hooks/{id}", settingsCtrl.SaveHook)
	mux.HandleFunc("DELETE /settings/hooks/{id}", settingsCtrl.DeleteHook)
	mux.HandleFunc("GET /settings/reports/{id}", settingsCtrl.EditReport)
	mux.HandleFunc("POST /settings/reports/{id}", settingsCtrl.SaveReport)
	mux.HandleFunc("DELETE /settings/reports/{id}", settingsCtrl.DeleteReport)
	mux.HandleFunc("GET /cases/{cid}/reports", settingsCtrl.ReportsDialog)
	mux.HandleFunc("POST /cases/{cid}/render", settingsCtrl.GenerateReport)

	// auditlog
	auditlogCtrl := handler.NewAuditlogCtrl(db, acl)
	mux.HandleFunc("GET /settings/auditlog/", auditlogCtrl.List)
	mux.HandleFunc("GET /settings/auditlog/{oid}", auditlogCtrl.ListForObject)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	eventCtrl := handler.NewEventCtrl(db, acl, ts)
	mux.HandleFunc("GET /cases/{cid}/events/", eventCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/events/export", eventCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/events/import", eventCtrl.ImportCSV)
	mux.HandleFunc("POST /cases/{cid}/events/import", eventCtrl.ImportCSV)
	mux.HandleFunc("POST /cases/{cid}/events/timesketch", eventCtrl.ImportTimesketch)
	mux.HandleFunc("GET /cases/{cid}/events/{id}", eventCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/events/{id}", eventCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/events/{id}", eventCtrl.Delete)

	// assets
	assetCtrl := handler.NewAssetCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/assets/", assetCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/assets/export", assetCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/assets/import", assetCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/assets/import", assetCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/assets/{id}", assetCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/assets/{id}", assetCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/assets/{id}", assetCtrl.Delete)

	// malware
	malwareCtrl := handler.NewMalwareCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/malware/", malwareCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/malware/export", malwareCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/malware/import", malwareCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/malware/import", malwareCtrl.Import)
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
	mux.HandleFunc("GET /cases/{cid}/evidences/export", evidenceCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/evidences/import", evidenceCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/evidences/import", evidenceCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}", evidenceCtrl.Edit)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}/download", evidenceCtrl.Download)
	mux.HandleFunc("POST /cases/{cid}/evidences/{id}", evidenceCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/evidences/{id}", evidenceCtrl.Delete)

	// evidence processing jobs
	jobCtrl := handler.NewJobCtrl(db, acl)
	mux.HandleFunc("GET /internal/jobs", jobCtrl.PopJob)
	mux.HandleFunc("POST /internal/jobs/ack", jobCtrl.AckJob)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}/run", jobCtrl.ListMods)
	mux.HandleFunc("POST /cases/{cid}/evidences/{id}/run", jobCtrl.PushJob)

	// tasks
	taskCtrl := handler.NewTaskCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/tasks/", taskCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/tasks/export", taskCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/tasks/import", taskCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/tasks/import", taskCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/tasks/{id}", taskCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/tasks/{id}", taskCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/tasks/{id}", taskCtrl.Delete)

	// notes
	noteCtrl := handler.NewNoteCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/notes/", noteCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/notes/export", noteCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/notes/import", noteCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/notes/import", noteCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/notes/{id}", noteCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/notes/{id}", noteCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/notes/{id}", noteCtrl.Delete)

	// visualizations
	visualsCtrl := handler.NewVisualsCtrl(db, acl)
	mux.HandleFunc("GET /cases/{cid}/vis/network", visualsCtrl.Network)
	mux.HandleFunc("GET /cases/{cid}/vis/timeline", visualsCtrl.Timeline)

	// --------------------------------------
	// Static Assets
	// --------------------------------------
	mux.Handle("GET /favicon.ico", handler.ServeFile(filepath.Join(cfg.AssetsFolder, "favicon.ico")))
	mux.Handle("GET /web/", handler.ServeDir("/web/", cfg.AssetsFolder))

	// --------------------------------------
	// Initialize Dagobert
	// --------------------------------------
	err = InitializeDagobert(db, acl, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dagobert: %v", err)
	}

	err = handler.LoadHooks(db)
	if err != nil {
		log.Fatalf("Failed to load hooks: %v", err)
	}

	err = db.RescheduleStaleJobs(handler.ServerToken)
	if err != nil {
		log.Fatalf("Failed to reschedule state jobs: %v", err)
	}

	log.Printf("Ready to receive requests. Listening on :8080 ...")
	err = http.ListenAndServe(":8080", srv)
	if err != nil {
		fmt.Printf("| %s | %v\n", tty.Red("ERR"), err)
	}
}

func InitializeDatabase(store *model.Store) error {
	db, err := sqlite.WithInstance(store.DB, &sqlite.Config{})
	if err != nil {
		return err
	}

	source, err := iofs.New(model.Migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", db)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	v, _, err := m.Version()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Printf("Migrated database to version %d", v)
	return nil
}

func InitializeDagobert(store *model.Store, acl *handler.ACL, cfg Configuration) error {
	users, err := store.ListUsers()
	if err != nil {
		return err
	}

	if len(users) == 0 {
		// initialize administrators
		log.Printf("Initializing administrators")
		for _, env := range os.Environ() {
			if !strings.HasPrefix(env, "DAGOBERT_ADMIN_") {
				continue
			}

			key, value, _ := strings.Cut(env, "=")
			log.Printf("  Adding %q as administrator", value)
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

	return nil
}
