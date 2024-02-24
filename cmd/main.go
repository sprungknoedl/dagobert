package main

import (
	"cmp"
	"net/url"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sprungknoedl/dagobert/internal/handler"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/oidc"
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
		AssetsFolder:   cmp.Or(os.Getenv("ASSETS_FOLDER"), "./web"),
		EvidenceFolder: cmp.Or(os.Getenv("EVIDENCE_FOLDER"), "./files/evidences"),
		Database:       cmp.Or(os.Getenv("DB_URL"), "./files/dagobert.db"),
		ClientId:       os.Getenv("CLIENT_ID"),
		ClientSecret:   os.Getenv("CLIENT_SECRET"),
		ClientUrl:      os.Getenv("CLIENT_URL"),
		Issuer:         os.Getenv("ISSUER"),
		SessionSecret:  os.Getenv("SESSION_SECRET"),
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

	// --------------------------------------
	// OIDC Authentication
	// --------------------------------------
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	e.Use(oidc.Middleware(e, oidc.Config{
		SessionName:   "default",
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
	err := handler.LoadTemplates("./files/templates/")
	if err != nil {
		e.Logger.Fatalf("failed to load report: %v", err)
	}

	// --------------------------------------
	// Home
	// --------------------------------------
	// cases
	caseCtrl := handler.NewCaseCtrl()
	e.GET("/", caseCtrl.ListCases).Name = "list-cases"
	e.GET("/cases/export", caseCtrl.ExportCases).Name = "export-cases"
	e.GET("/cases/import", caseCtrl.ImportCases).Name = "import-cases"
	e.POST("/cases/import", caseCtrl.ImportCases).Name = "import-cases"
	e.GET("/cases/:cid/show", caseCtrl.ShowCase).Name = "show-case"
	e.GET("/cases/:cid", caseCtrl.ViewCase).Name = "view-case"
	e.POST("/cases/:cid", caseCtrl.SaveCase).Name = "save-case"
	e.DELETE("/cases/:cid", caseCtrl.DeleteCase).Name = "delete-case"

	// templates
	reportCtrl := handler.NewReportCtrl()
	e.GET("/cases/:cid/reports", reportCtrl.ListTemplates).Name = "choose-report"
	e.GET("/cases/:cid/render", reportCtrl.ApplyTemplate).Name = "generate-report"

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	eventCtrl := handler.NewEventCtrl()
	e.GET("/cases/:cid/events", eventCtrl.ListEvents).Name = "list-events"
	e.GET("/cases/:cid/events/export", eventCtrl.ExportEvents).Name = "export-events"
	e.GET("/cases/:cid/events/import", eventCtrl.ImportEvents).Name = "import-events"
	e.POST("/cases/:cid/events/import", eventCtrl.ImportEvents).Name = "import-events"
	e.GET("/cases/:cid/events/:id/show", eventCtrl.ShowEvent).Name = "show-event"
	e.GET("/cases/:cid/events/:id", eventCtrl.ViewEvent).Name = "view-event"
	e.POST("/cases/:cid/events/:id", eventCtrl.SaveEvent).Name = "save-event"
	e.DELETE("/cases/:cid/events/:id", eventCtrl.DeleteEvent).Name = "delete-event"

	// assets
	assetCtrl := handler.NewAssetCtrl()
	e.GET("/cases/:cid/assets", assetCtrl.ListAssets).Name = "list-assets"
	e.GET("/cases/:cid/assets/export", assetCtrl.ExportAssets).Name = "export-assets"
	e.GET("/cases/:cid/assets/import", assetCtrl.ImportAssets).Name = "import-assets"
	e.POST("/cases/:cid/assets/import", assetCtrl.ImportAssets).Name = "import-assets"
	e.GET("/cases/:cid/assets/:id", assetCtrl.ViewAsset).Name = "view-asset"
	e.POST("/cases/:cid/assets/:id", assetCtrl.SaveAsset).Name = "save-asset"
	e.DELETE("/cases/:cid/assets/:id", assetCtrl.DeleteAsset).Name = "delete-asset"

	// malware
	malwareCtrl := handler.NewMalwareCtrl()
	e.GET("/cases/:cid/malware", malwareCtrl.ListMalware).Name = "list-malware"
	e.GET("/cases/:cid/malware.csv", malwareCtrl.ExportMalware).Name = "export-malware"
	e.GET("/cases/:cid/malware/import", malwareCtrl.ImportMalware).Name = "import-malware"
	e.POST("/cases/:cid/malware/import", malwareCtrl.ImportMalware).Name = "import-malware"
	e.GET("/cases/:cid/malware/:id", malwareCtrl.ViewMalware).Name = "view-malware"
	e.POST("/cases/:cid/malware/:id", malwareCtrl.SaveMalware).Name = "save-malware"
	e.DELETE("/cases/:cid/malware/:id", malwareCtrl.DeleteMalware).Name = "delete-malware"

	// indicators
	indicatorCtrl := handler.NewIndicatorCtrl()
	e.GET("/cases/:cid/indicators", indicatorCtrl.ListIndicators).Name = "list-indicators"
	e.GET("/cases/:cid/indicators.csv", indicatorCtrl.ExportIndicators).Name = "export-indicators"
	e.GET("/cases/:cid/indicators/import", indicatorCtrl.ImportIndicators).Name = "import-indicators"
	e.POST("/cases/:cid/indicators/import", indicatorCtrl.ImportIndicators).Name = "import-indicators"
	e.GET("/cases/:cid/indicators/:id", indicatorCtrl.ViewIndicator).Name = "view-indicator"
	e.POST("/cases/:cid/indicators/:id", indicatorCtrl.SaveIndicator).Name = "save-indicator"
	e.DELETE("/cases/:cid/indicators/:id", indicatorCtrl.DeleteIndicator).Name = "delete-indicator"

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// users
	userCtrl := handler.NewUserCtrl()
	e.GET("/users", userCtrl.ListUsers).Name = "list-users"
	e.GET("/users/:id", userCtrl.ViewUser).Name = "view-user"
	e.POST("/users/:id", userCtrl.SaveUser).Name = "save-user"
	e.DELETE("/users/:id", userCtrl.DeleteUser).Name = "delete-user"

	// evidence
	evidenceCtrl := handler.NewEvidenceCtrl()
	e.GET("/cases/:cid/evidences", evidenceCtrl.ListEvidences).Name = "list-evidences"
	e.GET("/cases/:cid/evidences/export", evidenceCtrl.ExportEvidences).Name = "export-evidences"
	e.GET("/cases/:cid/evidences/import", evidenceCtrl.ImportEvidences).Name = "import-evidences"
	e.POST("/cases/:cid/evidences/import", evidenceCtrl.ImportEvidences).Name = "import-evidences"
	e.GET("/cases/:cid/evidences/:id", evidenceCtrl.ViewEvidence).Name = "view-evidence"
	e.GET("/cases/:cid/evidences/:id/download", evidenceCtrl.DownloadEvidence).Name = "download-evidence"
	e.POST("/cases/:cid/evidences/:id", evidenceCtrl.SaveEvidence).Name = "save-evidence"
	e.DELETE("/cases/:cid/evidences/:id", evidenceCtrl.DeleteEvidence).Name = "delete-evidence"

	// tasks
	taskCtrl := handler.NewTaskCtrl()
	e.GET("/cases/:cid/tasks", taskCtrl.ListTasks).Name = "list-tasks"
	e.GET("/cases/:cid/tasks/export", taskCtrl.ExportTasks).Name = "export-tasks"
	e.GET("/cases/:cid/tasks/import", taskCtrl.ImportTasks).Name = "import-tasks"
	e.POST("/cases/:cid/tasks/import", taskCtrl.ImportTasks).Name = "import-tasks"
	e.GET("/cases/:cid/tasks/:id", taskCtrl.ViewTask).Name = "view-task"
	e.POST("/cases/:cid/tasks/:id", taskCtrl.SaveTask).Name = "save-task"
	e.DELETE("/cases/:cid/tasks/:id", taskCtrl.DeleteTask).Name = "delete-task"

	// notes
	noteCtrl := handler.NewNoteCtrl()
	e.GET("/cases/:cid/notes", noteCtrl.ListNotes).Name = "list-notes"
	e.GET("/cases/:cid/notes/export", noteCtrl.ExportNotes).Name = "export-notes"
	e.GET("/cases/:cid/notes/import", noteCtrl.ImportNotes).Name = "import-notes"
	e.POST("/cases/:cid/notes/import", noteCtrl.ImportNotes).Name = "import-notes"
	e.GET("/cases/:cid/notes/:id", noteCtrl.ViewNote).Name = "view-note"
	e.POST("/cases/:cid/notes/:id", noteCtrl.SaveNote).Name = "save-note"
	e.DELETE("/cases/:cid/notes/:id", noteCtrl.DeleteNote).Name = "delete-note"

	// --------------------------------------
	// Assets
	// --------------------------------------
	e.File("/favicon.ico", "dist/favicon.svg")
	e.Static("/dist", cfg.AssetsFolder)

	e.Logger.Fatal(e.Start(":8080"))
}
