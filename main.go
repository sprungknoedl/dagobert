package main

import (
	"cmp"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/extensions"
	"github.com/sprungknoedl/dagobert/internal/handler"
	"github.com/sprungknoedl/dagobert/internal/model"
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
	cfg := Configuration{
		AssetsFolder:   cmp.Or(os.Getenv("FS_ASSETS_FOLDER"), "./web"),
		EvidenceFolder: cmp.Or(os.Getenv("FS_EVIDENCE_FOLDER"), "./files/evidences"),
		Database:       cmp.Or(os.Getenv("DB_URL"), "./files/dagobert.db"),
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

	// --------------------------------------
	// Extensions
	// --------------------------------------
	err = extensions.Load()
	if err != nil {
		log.Fatalf("Failed to load extensions: %v", err)
	}

	// --------------------------------------
	// Authentication
	// --------------------------------------
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	auth := handler.NewAuthCtrl(db, handler.OpenIDConfig{
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
	caseCtrl := handler.NewCaseCtrl(db)
	mux.HandleFunc("GET /cases/", caseCtrl.List)
	mux.HandleFunc("GET /cases/export", caseCtrl.Export)
	mux.HandleFunc("GET /cases/import", caseCtrl.Import)
	mux.HandleFunc("POST /cases/import", caseCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}", caseCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}", caseCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}", caseCtrl.Delete)

	// users
	userCtrl := handler.NewUserCtrl(db)
	mux.HandleFunc("GET /settings/users/", userCtrl.List)
	mux.HandleFunc("GET /settings/users/{id}", userCtrl.Edit)
	mux.HandleFunc("POST /settings/users/{id}", userCtrl.Save)
	mux.HandleFunc("DELETE /settings/users/{id}", userCtrl.Delete)

	// api keys
	keyCtrl := handler.NewKeyCtrl(db)
	mux.HandleFunc("GET /settings/api-keys/", keyCtrl.List)
	mux.HandleFunc("GET /settings/api-keys/{key}", keyCtrl.Edit)
	mux.HandleFunc("POST /settings/api-keys/{key}", keyCtrl.Save)
	mux.HandleFunc("DELETE /settings/api-keys/{key}", keyCtrl.Delete)

	// templates
	reportCtrl := handler.NewReportCtrl(db)
	mux.HandleFunc("GET /settings/reports/", reportCtrl.List)
	mux.HandleFunc("GET /settings/reports/{id}", reportCtrl.Edit)
	mux.HandleFunc("POST /settings/reports/{id}", reportCtrl.Save)
	mux.HandleFunc("DELETE /settings/reports/{id}", reportCtrl.Delete)
	mux.HandleFunc("GET /cases/{cid}/reports", reportCtrl.Dialog)
	mux.HandleFunc("POST /cases/{cid}/render", reportCtrl.Generate)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	eventCtrl := handler.NewEventCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/events/", eventCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/events/export", eventCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/events/import", eventCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/events/import", eventCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/events/{id}", eventCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/events/{id}", eventCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/events/{id}", eventCtrl.Delete)

	// assets
	assetCtrl := handler.NewAssetCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/assets/", assetCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/assets/export", assetCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/assets/import", assetCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/assets/import", assetCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/assets/{id}", assetCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/assets/{id}", assetCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/assets/{id}", assetCtrl.Delete)

	// malware
	malwareCtrl := handler.NewMalwareCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/malware/", malwareCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/malware/export", malwareCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/malware/import", malwareCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/malware/import", malwareCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/malware/{id}", malwareCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/malware/{id}", malwareCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/malware/{id}", malwareCtrl.Delete)

	// indicators
	indicatorCtrl := handler.NewIndicatorCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/indicators/", indicatorCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/indicators/export", indicatorCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/indicators/import", indicatorCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/indicators/import", indicatorCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/indicators/{id}", indicatorCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/indicators/{id}", indicatorCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/indicators/{id}", indicatorCtrl.Delete)

	// evidence
	evidenceCtrl := handler.NewEvidenceCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/evidences/", evidenceCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/evidences/export", evidenceCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/evidences/import", evidenceCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/evidences/import", evidenceCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}", evidenceCtrl.Edit)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}/download", evidenceCtrl.Download)
	mux.HandleFunc("GET /cases/{cid}/evidences/{id}/run", evidenceCtrl.Extensions)
	mux.HandleFunc("POST /cases/{cid}/evidences/{id}/run", evidenceCtrl.Run)
	mux.HandleFunc("POST /cases/{cid}/evidences/{id}", evidenceCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/evidences/{id}", evidenceCtrl.Delete)

	// tasks
	taskCtrl := handler.NewTaskCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/tasks/", taskCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/tasks/export", taskCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/tasks/import", taskCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/tasks/import", taskCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/tasks/{id}", taskCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/tasks/{id}", taskCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/tasks/{id}", taskCtrl.Delete)

	// notes
	noteCtrl := handler.NewNoteCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/notes/", noteCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/notes/export", noteCtrl.Export)
	mux.HandleFunc("GET /cases/{cid}/notes/import", noteCtrl.Import)
	mux.HandleFunc("POST /cases/{cid}/notes/import", noteCtrl.Import)
	mux.HandleFunc("GET /cases/{cid}/notes/{id}", noteCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}/notes/{id}", noteCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}/notes/{id}", noteCtrl.Delete)

	// --------------------------------------
	// Static Assets
	// --------------------------------------
	mux.Handle("GET /favicon.ico", handler.ServeFile("dist/favicon.ico"))
	mux.Handle("GET /dist/", handler.ServeDir("/dist/", cfg.AssetsFolder))

	// --------------------------------------
	// Initialize Dagobert
	// --------------------------------------
	err = InitializeDagobert(db, auth, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dagobert: %v", err)
	}

	log.Printf("Ready to receive requests. Listening on :8080 ...")
	err = http.ListenAndServe(":8080", srv)
	if err != nil {
		fmt.Printf("| %s | %v\n", tty.Red("ERR"), err)
	}
}

func InitializeDagobert(store *model.Store, auth *handler.AuthCtrl, cfg Configuration) error {
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

			err = auth.SaveRoleAssignment(value, "Administrator")
			if err != nil {
				return err
			}
		}
	}

	return nil
}
