package main

import (
	"cmp"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/sprungknoedl/dagobert/internal/handler"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
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

	Superadmin string
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
		Superadmin:     os.Getenv("DAGOBERT_ADMIN"),
	}

	db, err := model.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = InitializeDagobert(db, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dagobert: %v", err)
	}

	// --------------------------------------
	// Authentication
	// --------------------------------------
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	userCtrl := handler.NewUserCtrl(db, handler.OpenIDConfig{
		ClientId:      cfg.ClientId,
		ClientSecret:  cfg.ClientSecret,
		Issuer:        *issuer,
		ClientUrl:     *clientUrl,
		Identifier:    cfg.IdentityClaim,
		Scopes:        []string{"openid", "profile", "email"},
		PostLogoutUrl: *clientUrl,
	})

	// --------------------------------------
	// Router
	// --------------------------------------
	mux := http.NewServeMux()
	srv := Recover(mux)
	srv = Logger(srv)
	srv = userCtrl.Protect(srv)

	// --------------------------------------
	// Reports
	// --------------------------------------
	err = handler.LoadTemplates("./files/templates/")
	if err != nil {
		log.Fatalf("failed to load report: %v", err)
	}

	// --------------------------------------
	// Home
	// --------------------------------------
	// index
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cases/", http.StatusTemporaryRedirect)
	})

	// cases
	caseCtrl := handler.NewCaseCtrl(db)
	mux.HandleFunc("GET /cases/", caseCtrl.List)
	mux.HandleFunc("GET /cases/export", caseCtrl.Export)
	mux.HandleFunc("GET /cases/import", caseCtrl.ImportCases)
	mux.HandleFunc("POST /cases/import", caseCtrl.ImportCases)
	mux.HandleFunc("GET /cases/{cid}", caseCtrl.Edit)
	mux.HandleFunc("POST /cases/{cid}", caseCtrl.Save)
	mux.HandleFunc("DELETE /cases/{cid}", caseCtrl.Delete)

	// users / authentication
	mux.HandleFunc("GET /users/", userCtrl.List)
	mux.HandleFunc("GET /logout", userCtrl.Logout)
	mux.HandleFunc("GET /oidc-callback", userCtrl.Callback)

	// templates
	reportCtrl := handler.NewReportCtrl(db)
	mux.HandleFunc("GET /cases/{cid}/reports", reportCtrl.List)
	mux.HandleFunc("GET /cases/{cid}/render", reportCtrl.Generate)

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
	// Assets
	// --------------------------------------
	mux.Handle("GET /favicon.ico", utils.ServeFile("dist/favicon.ico"))
	mux.Handle("GET /dist/", utils.ServeDir("/dist/", cfg.AssetsFolder))

	err = http.ListenAndServe(":8080", srv)
	if err != nil {
		fmt.Printf("| %s | %v\n", tty.Red("ERR"), err)
	}
}

func InitializeDagobert(store *model.Store, cfg Configuration) error {
	users, err := store.FindUsers("", "")
	if err != nil {
		return err
	}

	if len(users) == 0 && cfg.Superadmin != "" {
		// initialize super user
		log.Printf("Initializing super user ...")
		err = store.SaveUser(model.User{
			ID: cfg.Superadmin,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
