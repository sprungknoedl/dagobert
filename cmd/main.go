package main

import (
	"encoding/gob"
	"net/url"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sprungknoedl/dagobert/components/utils"
	"github.com/sprungknoedl/dagobert/handler"
	"github.com/sprungknoedl/dagobert/model"
)

type Configuration struct {
	AssetsFolder   string
	EvidenceFolder string

	Database string

	ClientId     string
	ClientSecret string
	Issuer       string
	ClientUrl    string

	SessionSecret string
}

func main() {
	cfg := Configuration{
		AssetsFolder:   getEnv("ASSETS_FOLDER", "./dist"),
		EvidenceFolder: getEnv("EVIDENCE_FOLDER", "./files/evidences"),

		Database: getEnv("DB_URL", "./dagobert.db"),

		ClientId:     getEnv("CLIENT_ID", ""),
		ClientSecret: getEnv("CLIENT_SECRET", ""),
		ClientUrl:    getEnv("CLIENT_URL", ""),
		Issuer:       getEnv("ISSUER", ""),

		SessionSecret: getEnv("SESSION_SECRET", ""),
	}

	model.InitDatabase(cfg.Database)

	e := echo.New()
	e.HTTPErrorHandler = handler.ErrorHandler
	e.Use(PrettyLogger)
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	// --------------------------------------
	// Session Store
	// --------------------------------------
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	e.Use(session.Middleware(store))
	gob.Register(utils.CaseDTO{})

	// --------------------------------------
	// OIDC Authentication
	// --------------------------------------
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	e.Use(OIDC(e, InitParams{
		ClientId:      cfg.ClientId,
		ClientSecret:  cfg.ClientSecret,
		Issuer:        *issuer,
		ClientUrl:     *clientUrl,
		Scopes:        []string{"openid", "profile", "email"},
		PostLogoutUrl: *clientUrl,
	}))

	// --------------------------------------
	// Reports
	// --------------------------------------
	err := handler.LoadTemplates("./templates/")
	if err != nil {
		e.Logger.Fatalf("failed to load report: %v", err)
	}

	// --------------------------------------
	// Home
	// --------------------------------------
	// cases
	e.GET("/", handler.ListCases).Name = "list-cases"
	e.GET("/cases/export", handler.ExportCases).Name = "export-cases"
	e.GET("/cases/import", handler.ImportCases).Name = "import-cases"
	e.POST("/cases/import", handler.ImportCases).Name = "import-cases"
	e.GET("/cases/:cid/show", handler.ShowCase).Name = "show-case"
	e.GET("/cases/:cid", handler.ViewCase).Name = "view-case"
	e.POST("/cases/:cid", handler.SaveCase).Name = "save-case"
	e.DELETE("/cases/:cid", handler.DeleteCase).Name = "delete-case"

	// templates
	e.GET("/cases/:cid/reports", handler.ListTemplates).Name = "choose-report"
	e.GET("/cases/:cid/render", handler.ApplyTemplate).Name = "generate-report"

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	e.GET("/cases/:cid/events", handler.ListEvents).Name = "list-events"
	e.GET("/cases/:cid/events/export", handler.ExportEvents).Name = "export-events"
	e.GET("/cases/:cid/events/import", handler.ImportEvents).Name = "import-events"
	e.POST("/cases/:cid/events/import", handler.ImportEvents).Name = "import-events"
	e.GET("/cases/:cid/events/:id", handler.ViewEvent).Name = "view-event"
	e.POST("/cases/:cid/events/:id", handler.SaveEvent).Name = "save-event"
	e.DELETE("/cases/:cid/events/:id", handler.DeleteEvent).Name = "delete-event"

	// assets
	e.GET("/cases/:cid/assets", handler.ListAssets).Name = "list-assets"
	e.GET("/cases/:cid/assets/export", handler.ExportAssets).Name = "export-assets"
	e.GET("/cases/:cid/assets/import", handler.ImportAssets).Name = "import-assets"
	e.POST("/cases/:cid/assets/import", handler.ImportAssets).Name = "import-assets"
	e.GET("/cases/:cid/assets/:id", handler.ViewAsset).Name = "view-asset"
	e.POST("/cases/:cid/assets/:id", handler.SaveAsset).Name = "save-asset"
	e.DELETE("/cases/:cid/assets/:id", handler.DeleteAsset).Name = "delete-asset"

	// malware
	e.GET("/cases/:cid/malware", handler.ListMalware).Name = "list-malware"
	e.GET("/cases/:cid/malware.csv", handler.ExportMalware).Name = "export-malware"
	e.GET("/cases/:cid/malware/import", handler.ImportMalware).Name = "import-malware"
	e.POST("/cases/:cid/malware/import", handler.ImportMalware).Name = "import-malware"
	e.GET("/cases/:cid/malware/:id", handler.ViewMalware).Name = "view-malware"
	e.POST("/cases/:cid/malware/:id", handler.SaveMalware).Name = "save-malware"
	e.DELETE("/cases/:cid/malware/:id", handler.DeleteMalware).Name = "delete-malware"

	// indicators
	e.GET("/cases/:cid/indicators", handler.ListIndicators).Name = "list-indicators"
	e.GET("/cases/:cid/indicators.csv", handler.ExportIndicators).Name = "export-indicators"
	e.GET("/cases/:cid/indicators/import", handler.ImportIndicators).Name = "import-indicators"
	e.POST("/cases/:cid/indicators/import", handler.ImportIndicators).Name = "import-indicators"
	e.GET("/cases/:cid/indicators/:id", handler.ViewIndicator).Name = "view-indicator"
	e.POST("/cases/:cid/indicators/:id", handler.SaveIndicator).Name = "save-indicator"
	e.DELETE("/cases/:cid/indicators/:id", handler.DeleteIndicator).Name = "delete-indicator"

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// users
	e.GET("/cases/:cid/users", handler.ListUsers).Name = "list-users"
	e.GET("/cases/:cid/users/export", handler.ExportUsers).Name = "export-users"
	e.GET("/cases/:cid/users/import", handler.ImportUsers).Name = "import-users"
	e.POST("/cases/:cid/users/import", handler.ImportUsers).Name = "import-users"
	e.GET("/cases/:cid/users/:id", handler.ViewUser).Name = "view-user"
	e.POST("/cases/:cid/users/:id", handler.SaveUser).Name = "save-user"
	e.DELETE("/cases/:cid/users/:id", handler.DeleteUser).Name = "delete-user"

	// evidence
	e.GET("/cases/:cid/evidences", handler.ListEvidences).Name = "list-evidences"
	e.GET("/cases/:cid/evidences/export", handler.ExportEvidences).Name = "export-evidences"
	e.GET("/cases/:cid/evidences/import", handler.ImportEvidences).Name = "import-evidences"
	e.POST("/cases/:cid/evidences/import", handler.ImportEvidences).Name = "import-evidences"
	e.GET("/cases/:cid/evidences/:id", handler.ViewEvidence).Name = "view-evidence"
	e.GET("/cases/:cid/evidences/:id/download", handler.DownloadEvidence).Name = "download-evidence"
	e.POST("/cases/:cid/evidences/:id", handler.SaveEvidence).Name = "save-evidence"
	e.DELETE("/cases/:cid/evidences/:id", handler.DeleteEvidence).Name = "delete-evidence"

	// tasks
	e.GET("/cases/:cid/tasks", handler.ListTasks).Name = "list-tasks"
	e.GET("/cases/:cid/tasks/export", handler.ExportTasks).Name = "export-tasks"
	e.GET("/cases/:cid/tasks/import", handler.ImportTasks).Name = "import-tasks"
	e.POST("/cases/:cid/tasks/import", handler.ImportTasks).Name = "import-tasks"
	e.GET("/cases/:cid/tasks/:id", handler.ViewTask).Name = "view-task"
	e.POST("/cases/:cid/tasks/:id", handler.SaveTask).Name = "save-task"
	e.DELETE("/cases/:cid/tasks/:id", handler.DeleteTask).Name = "delete-task"

	// notes
	e.GET("/cases/:cid/notes", handler.ListNotes).Name = "list-notes"
	e.GET("/cases/:cid/notes/export", handler.ExportNotes).Name = "export-notes"
	e.GET("/cases/:cid/notes/import", handler.ImportNotes).Name = "import-notes"
	e.POST("/cases/:cid/notes/import", handler.ImportNotes).Name = "import-notes"
	e.GET("/cases/:cid/notes/:id", handler.ViewNote).Name = "view-note"
	e.POST("/cases/:cid/notes/:id", handler.SaveNote).Name = "save-note"
	e.DELETE("/cases/:cid/notes/:id", handler.DeleteNote).Name = "delete-note"

	// --------------------------------------
	// Assets
	// --------------------------------------
	e.File("/favicon.ico", "dist/favicon.svg")
	e.Static("/dist", cfg.AssetsFolder)

	e.Logger.Fatal(e.Start(":8080"))
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
