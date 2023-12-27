package main

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	gin_oidc "github.com/maximRnback/gin-oidc"
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

var cfg = Configuration{}

func main() {
	cfg = Configuration{
		AssetsFolder:   getEnv("ASSETS_FOLDER", "./dist"),
		EvidenceFolder: getEnv("EVIDENCE_FOLDER", "./files/evidences"),

		Database: getEnv("DB_URL", "./dagobert.db"),

		ClientId:     getEnv("CLIENT_ID", ""),
		ClientSecret: getEnv("CLIENT_SECRET", ""),
		ClientUrl:    getEnv("CLIENT_URL", ""),
		Issuer:       getEnv("ISSUER", ""),

		SessionSecret: getEnv("SESSION_SECRET", ""),
	}
	log.Printf("configuration: %+v", cfg)

	model.InitDatabase(cfg.Database)

	r := gin.Default()

	// --------------------------------------
	// Session Store
	// --------------------------------------
	store := cookie.NewStore([]byte(cfg.SessionSecret))
	r.Use(sessions.Sessions("dagobert", store))

	// --------------------------------------
	// OIDC Authentication
	// --------------------------------------
	issuer, _ := url.Parse(cfg.Issuer)
	clientUrl, _ := url.Parse(cfg.ClientUrl)
	r.Use(gin_oidc.Init(gin_oidc.InitParams{
		Router:       r,
		ClientId:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		Issuer:       *issuer,
		ClientUrl:    *clientUrl,
		Scopes:       []string{"openid", "profile", "email"},
		ErrorHandler: func(c *gin.Context) {
			message := c.Errors.Last().Error()
			c.String(http.StatusExpectationFailed, "oidc: %s", message)
		},
		PostLogoutUrl: *clientUrl,
	}))

	// --------------------------------------
	// Home
	// --------------------------------------
	r.StaticFile("/", "dist/index.html")
	r.StaticFile("/favicon.ico", "dist/favicon.svg")

	// cases
	r.GET("/api/cases", handler.ListCaseR)
	r.GET("/api/cases.csv", handler.ExportCaseCsvR)
	r.GET("/api/cases/:cid", handler.GetCaseR)
	r.POST("/api/cases", handler.AddCaseR)
	r.PUT("/api/cases/:cid", handler.EditCaseR)
	r.DELETE("/api/cases/:cid", handler.DeleteCaseR)

	// templates
	r.GET("/api/templates", handler.ListTemplateR)
	r.GET("/api/cases/:cid/render", handler.ApplyTemplateR)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	r.GET("/api/cases/:cid/events", handler.ListEventR)
	r.GET("/api/cases/:cid/events.csv", handler.ExportEventCsvR)
	r.GET("/api/cases/:cid/events/:id", handler.GetEventR)
	r.POST("/api/cases/:cid/events", handler.AddEventR)
	r.PUT("/api/cases/:cid/events/:id", handler.EditEventR)
	r.DELETE("/api/cases/:cid/events/:id", handler.DeleteEventR)

	// assets
	r.GET("/api/cases/:cid/assets", handler.ListAssetR)
	r.GET("/api/cases/:cid/assets.csv", handler.ExportAssetCsvR)
	r.GET("/api/cases/:cid/assets/:id", handler.GetAssetR)
	r.POST("/api/cases/:cid/assets", handler.AddAssetR)
	r.PUT("/api/cases/:cid/assets/:id", handler.EditAssetR)
	r.DELETE("/api/cases/:cid/assets/:id", handler.DeleteAssetR)

	// malware
	r.GET("/api/cases/:cid/malware", handler.ListMalwareR)
	r.GET("/api/cases/:cid/malware.csv", handler.ExportMalwareCsvR)
	r.GET("/api/cases/:cid/malware/:id", handler.GetMalwareR)
	r.POST("/api/cases/:cid/malware", handler.AddMalwareR)
	r.PUT("/api/cases/:cid/malware/:id", handler.EditMalwareR)
	r.DELETE("/api/cases/:cid/malware/:id", handler.DeleteMalwareR)

	// indicators
	r.GET("/api/cases/:cid/indicators", handler.ListIndicatorR)
	r.GET("/api/cases/:cid/indicators.csv", handler.ExportIndicatorCsvR)
	r.GET("/api/cases/:cid/indicators/:id", handler.GetIndicatorR)
	r.POST("/api/cases/:cid/indicators", handler.AddIndicatorR)
	r.PUT("/api/cases/:cid/indicators/:id", handler.EditIndicatorR)
	r.DELETE("/api/cases/:cid/indicators/:id", handler.DeleteIndicatorR)

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// users
	r.GET("/api/cases/:cid/users", handler.ListUserR)
	r.GET("/api/cases/:cid/users.csv", handler.ExportUserCsvR)
	r.GET("/api/cases/:cid/users/:id", handler.GetUserR)
	r.POST("/api/cases/:cid/users", handler.AddUserR)
	r.PUT("/api/cases/:cid/users/:id", handler.EditUserR)
	r.DELETE("/api/cases/:cid/users/:id", handler.DeleteUserR)

	// evidence
	r.GET("/api/cases/:cid/evidences", handler.ListEvidenceR)
	r.GET("/api/cases/:cid/evidences.csv", handler.ExportEvidenceCsvR)
	r.GET("/api/cases/:cid/evidences/:id", handler.GetEvidenceR)
	r.POST("/api/cases/:cid/evidences", handler.AddEvidenceR)
	r.PUT("/api/cases/:cid/evidences/:id", handler.EditEvidenceR)
	r.DELETE("/api/cases/:cid/evidences/:id", handler.DeleteEvidenceR)

	// tasks
	r.GET("/api/cases/:cid/tasks", handler.ListTaskR)
	r.GET("/api/cases/:cid/tasks.csv", handler.ExportTaskCsvR)
	r.GET("/api/cases/:cid/tasks/:id", handler.GetTaskR)
	r.POST("/api/cases/:cid/tasks", handler.AddTaskR)
	r.PUT("/api/cases/:cid/tasks/:id", handler.EditTaskR)
	r.DELETE("/api/cases/:cid/tasks/:id", handler.DeleteTaskR)

	// notes
	r.GET("/api/cases/:cid/notes", handler.ListNoteR)
	r.GET("/api/cases/:cid/notes.csv", handler.ExportNoteCsvR)
	r.GET("/api/cases/:cid/notes/:id", handler.GetNoteR)
	r.POST("/api/cases/:cid/notes", handler.AddNoteR)
	r.PUT("/api/cases/:cid/notes/:id", handler.EditNoteR)
	r.DELETE("/api/cases/:cid/notes/:id", handler.DeleteNoteR)

	// --------------------------------------
	// Assets
	// --------------------------------------
	r.Static("/dist", cfg.AssetsFolder)
	r.Run()
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Printf("ENV | %s = %s", key, value)
		return value
	}
	return fallback
}
