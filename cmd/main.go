package main

import (
	"cmp"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sprungknoedl/dagobert/internal/handler"
	"github.com/sprungknoedl/dagobert/internal/sqlite"
	"github.com/sprungknoedl/dagobert/internal/templ/utils"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

const SessionName = "default"

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

	db, err := sqlite.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = InitializeDagobert(db, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dagobert: %v", err)
	}

	e := echo.New()
	e.HTTPErrorHandler = handler.ErrorHandler
	e.Use(PrettyLogger)
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())

	// --------------------------------------
	// Session Store
	// --------------------------------------
	e.Use(session.Middleware(
		sessions.NewCookieStore([]byte(cfg.SessionSecret))))

	// --------------------------------------
	// Authentication
	// --------------------------------------
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	userCtrl := handler.NewUserCtrl(db, handler.OpenIDConfig{
		SessionName:   SessionName,
		ClientId:      cfg.ClientId,
		ClientSecret:  cfg.ClientSecret,
		Issuer:        *issuer,
		ClientUrl:     *clientUrl,
		Identifier:    cfg.IdentityClaim,
		Scopes:        []string{"openid", "profile", "email"},
		PostLogoutUrl: *clientUrl,
	})
	e.Use(userCtrl.Protect(e))

	e.Use(InjectCase(db))
	e.Use(InjectUser(SessionName))

	// --------------------------------------
	// Reports
	// --------------------------------------
	err = handler.LoadTemplates("./files/templates/")
	if err != nil {
		e.Logger.Fatalf("failed to load report: %v", err)
	}

	// --------------------------------------
	// Home
	// --------------------------------------
	// cases
	caseCtrl := handler.NewCaseCtrl(db)
	e.GET("/", caseCtrl.List).Name = "list-cases"
	e.GET("/cases/export", caseCtrl.Export).Name = "export-cases"
	e.GET("/cases/import", caseCtrl.ImportCases).Name = "import-cases"
	e.POST("/cases/import", caseCtrl.ImportCases).Name = "import-cases"
	e.GET("/cases/:cid", caseCtrl.Edit).Name = "view-case"
	e.POST("/cases/:cid", caseCtrl.Save).Name = "save-case"
	e.DELETE("/cases/:cid", caseCtrl.Delete).Name = "delete-case"

	// templates
	reportCtrl := handler.NewReportCtrl(db)
	e.GET("/cases/:cid/reports", reportCtrl.List).Name = "choose-report"
	e.GET("/cases/:cid/render", reportCtrl.Generate).Name = "generate-report"

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	eventCtrl := handler.NewEventCtrl(db, db, db)
	e.GET("/cases/:cid/events", eventCtrl.List).Name = "list-events"
	e.GET("/cases/:cid/events/export", eventCtrl.Export).Name = "export-events"
	e.GET("/cases/:cid/events/import", eventCtrl.Import).Name = "import-events"
	e.POST("/cases/:cid/events/import", eventCtrl.Import).Name = "import-events"
	e.GET("/cases/:cid/events/:id", eventCtrl.Edit).Name = "view-event"
	e.POST("/cases/:cid/events/:id", eventCtrl.Save).Name = "save-event"
	e.DELETE("/cases/:cid/events/:id", eventCtrl.Delete).Name = "delete-event"

	// assets
	assetCtrl := handler.NewAssetCtrl(db)
	e.GET("/cases/:cid/assets", assetCtrl.List).Name = "list-assets"
	e.GET("/cases/:cid/assets/export", assetCtrl.Export).Name = "export-assets"
	e.GET("/cases/:cid/assets/import", assetCtrl.Import).Name = "import-assets"
	e.POST("/cases/:cid/assets/import", assetCtrl.Import).Name = "import-assets"
	e.GET("/cases/:cid/assets/:id", assetCtrl.Edit).Name = "view-asset"
	e.POST("/cases/:cid/assets/:id", assetCtrl.Save).Name = "save-asset"
	e.DELETE("/cases/:cid/assets/:id", assetCtrl.Delete).Name = "delete-asset"

	// malware
	malwareCtrl := handler.NewMalwareCtrl(db)
	e.GET("/cases/:cid/malware", malwareCtrl.List).Name = "list-malware"
	e.GET("/cases/:cid/malware.csv", malwareCtrl.Export).Name = "export-malware"
	e.GET("/cases/:cid/malware/import", malwareCtrl.Import).Name = "import-malware"
	e.POST("/cases/:cid/malware/import", malwareCtrl.Import).Name = "import-malware"
	e.GET("/cases/:cid/malware/:id", malwareCtrl.View).Name = "view-malware"
	e.POST("/cases/:cid/malware/:id", malwareCtrl.Save).Name = "save-malware"
	e.DELETE("/cases/:cid/malware/:id", malwareCtrl.Delete).Name = "delete-malware"

	// indicators
	indicatorCtrl := handler.NewIndicatorCtrl(db)
	e.GET("/cases/:cid/indicators", indicatorCtrl.List).Name = "list-indicators"
	e.GET("/cases/:cid/indicators.csv", indicatorCtrl.Export).Name = "export-indicators"
	e.GET("/cases/:cid/indicators/import", indicatorCtrl.Import).Name = "import-indicators"
	e.POST("/cases/:cid/indicators/import", indicatorCtrl.Import).Name = "import-indicators"
	e.GET("/cases/:cid/indicators/:id", indicatorCtrl.Edit).Name = "view-indicator"
	e.POST("/cases/:cid/indicators/:id", indicatorCtrl.Save).Name = "save-indicator"
	e.DELETE("/cases/:cid/indicators/:id", indicatorCtrl.Delete).Name = "delete-indicator"

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// evidence
	evidenceCtrl := handler.NewEvidenceCtrl(db)
	e.GET("/cases/:cid/evidences", evidenceCtrl.List).Name = "list-evidences"
	e.GET("/cases/:cid/evidences/export", evidenceCtrl.Export).Name = "export-evidences"
	e.GET("/cases/:cid/evidences/import", evidenceCtrl.Import).Name = "import-evidences"
	e.POST("/cases/:cid/evidences/import", evidenceCtrl.Import).Name = "import-evidences"
	e.GET("/cases/:cid/evidences/:id", evidenceCtrl.Edit).Name = "view-evidence"
	e.GET("/cases/:cid/evidences/:id/download", evidenceCtrl.Download).Name = "download-evidence"
	e.POST("/cases/:cid/evidences/:id", evidenceCtrl.Save).Name = "save-evidence"
	e.DELETE("/cases/:cid/evidences/:id", evidenceCtrl.Delete).Name = "delete-evidence"

	// tasks
	taskCtrl := handler.NewTaskCtrl(db)
	e.GET("/cases/:cid/tasks", taskCtrl.List).Name = "list-tasks"
	e.GET("/cases/:cid/tasks/export", taskCtrl.Export).Name = "export-tasks"
	e.GET("/cases/:cid/tasks/import", taskCtrl.Import).Name = "import-tasks"
	e.POST("/cases/:cid/tasks/import", taskCtrl.Import).Name = "import-tasks"
	e.GET("/cases/:cid/tasks/:id", taskCtrl.Edit).Name = "view-task"
	e.POST("/cases/:cid/tasks/:id", taskCtrl.Save).Name = "save-task"
	e.DELETE("/cases/:cid/tasks/:id", taskCtrl.Delete).Name = "delete-task"

	// notes
	noteCtrl := handler.NewNoteCtrl(db)
	e.GET("/cases/:cid/notes", noteCtrl.List).Name = "list-notes"
	e.GET("/cases/:cid/notes/export", noteCtrl.Export).Name = "export-notes"
	e.GET("/cases/:cid/notes/import", noteCtrl.Import).Name = "import-notes"
	e.POST("/cases/:cid/notes/import", noteCtrl.Import).Name = "import-notes"
	e.GET("/cases/:cid/notes/:id", noteCtrl.View).Name = "view-note"
	e.POST("/cases/:cid/notes/:id", noteCtrl.Save).Name = "save-note"
	e.DELETE("/cases/:cid/notes/:id", noteCtrl.Delete).Name = "delete-note"

	// --------------------------------------
	// Settings
	// --------------------------------------
	// users
	e.GET("/users", userCtrl.List).Name = "list-users"

	// --------------------------------------
	// Assets
	// --------------------------------------
	e.File("/favicon.ico", "dist/favicon.svg")
	e.Static("/dist", cfg.AssetsFolder)

	e.Logger.Fatal(e.Start(":8080"))
}

func InitializeDagobert(store model.UserStore, cfg Configuration) error {
	users, err := store.ListUsers()
	if err != nil {
		return err
	}

	if len(users) == 0 && cfg.Superadmin != "" {
		// initialize super user
		log.Printf("Initializing super user ...")
		_, err = store.SaveUser(model.User{
			ID: cfg.Superadmin,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func GetActiveCase(store model.CaseStore, c echo.Context) (model.Case, error) {
	cid, err := strconv.ParseInt(c.Param("cid"), 10, 64)
	if err != nil || cid == 0 {
		return model.Case{}, err
	}

	obj, err := store.GetCase(cid)
	return obj, err
}

func InjectCase(store model.CaseStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("case", utils.CaseDTO{})
			obj, err := GetActiveCase(store, c)
			if err == nil {
				c.Set("case", utils.CaseDTO{
					ID:   obj.ID,
					Name: obj.Name,
				})
			}

			return next(c)
		}
	}
}

func InjectUser(sessionName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", "unknown")

			sess, _ := session.Get(sessionName, c)
			claims, _ := sess.Values["oidcClaims"].(map[string]interface{})
			if sub, ok := claims["sub"].(string); ok {
				c.Set("user", sub)
			}

			return next(c)
		}
	}
}
