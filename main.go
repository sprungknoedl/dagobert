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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

var db *gorm.DB
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

	initDatabase()

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
	r.GET("/api/cases", ListCaseR)
	r.GET("/api/cases.csv", ExportCaseCsvR)
	r.GET("/api/cases/:cid", GetCaseR)
	r.POST("/api/cases", AddCaseR)
	r.PUT("/api/cases/:cid", EditCaseR)
	r.DELETE("/api/cases/:cid", DeleteCaseR)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	r.GET("/api/cases/:cid/events", ListEventR)
	r.GET("/api/cases/:cid/events.csv", ExportEventCsvR)
	r.GET("/api/cases/:cid/events/:id", GetEventR)
	r.POST("/api/cases/:cid/events", AddEventR)
	r.PUT("/api/cases/:cid/events/:id", EditEventR)
	r.DELETE("/api/cases/:cid/events/:id", DeleteEventR)

	// assets
	r.GET("/api/cases/:cid/assets", ListAssetR)
	r.GET("/api/cases/:cid/assets.csv", ExportAssetCsvR)
	r.GET("/api/cases/:cid/assets/:id", GetAssetR)
	r.POST("/api/cases/:cid/assets", AddAssetR)
	r.PUT("/api/cases/:cid/assets/:id", EditAssetR)
	r.DELETE("/api/cases/:cid/assets/:id", DeleteAssetR)

	// malware
	r.GET("/api/cases/:cid/malware", ListMalwareR)
	r.GET("/api/cases/:cid/malware.csv", ExportMalwareCsvR)
	r.GET("/api/cases/:cid/malware/:id", GetMalwareR)
	r.POST("/api/cases/:cid/malware", AddMalwareR)
	r.PUT("/api/cases/:cid/malware/:id", EditMalwareR)
	r.DELETE("/api/cases/:cid/malware/:id", DeleteMalwareR)

	// indicators
	r.GET("/api/cases/:cid/indicators", ListIndicatorR)
	r.GET("/api/cases/:cid/indicators.csv", ExportIndicatorCsvR)
	r.GET("/api/cases/:cid/indicators/:id", GetIndicatorR)
	r.POST("/api/cases/:cid/indicators", AddIndicatorR)
	r.PUT("/api/cases/:cid/indicators/:id", EditIndicatorR)
	r.DELETE("/api/cases/:cid/indicators/:id", DeleteIndicatorR)

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// users
	r.GET("/api/cases/:cid/users", ListUserR)
	r.GET("/api/cases/:cid/users.csv", ExportUserCsvR)
	r.GET("/api/cases/:cid/users/:id", GetUserR)
	r.POST("/api/cases/:cid/users", AddUserR)
	r.PUT("/api/cases/:cid/users/:id", EditUserR)
	r.DELETE("/api/cases/:cid/users/:id", DeleteUserR)

	// evidence
	r.GET("/api/cases/:cid/evidences", ListEvidenceR)
	r.GET("/api/cases/:cid/evidences.csv", ExportEvidenceCsvR)
	r.GET("/api/cases/:cid/evidences/:id", GetEvidenceR)
	r.POST("/api/cases/:cid/evidences", AddEvidenceR)
	r.PUT("/api/cases/:cid/evidences/:id", EditEvidenceR)
	r.DELETE("/api/cases/:cid/evidences/:id", DeleteEvidenceR)

	// tasks
	r.GET("/api/cases/:cid/tasks", ListTaskR)
	r.GET("/api/cases/:cid/tasks.csv", ExportTaskCsvR)
	r.GET("/api/cases/:cid/tasks/:id", GetTaskR)
	r.POST("/api/cases/:cid/tasks", AddTaskR)
	r.PUT("/api/cases/:cid/tasks/:id", EditTaskR)
	r.DELETE("/api/cases/:cid/tasks/:id", DeleteTaskR)

	// notes
	r.GET("/api/cases/:cid/notes", ListNoteR)
	r.GET("/api/cases/:cid/notes.csv", ExportNoteCsvR)
	r.GET("/api/cases/:cid/notes/:id", GetNoteR)
	r.POST("/api/cases/:cid/notes", AddNoteR)
	r.PUT("/api/cases/:cid/notes/:id", EditNoteR)
	r.DELETE("/api/cases/:cid/notes/:id", DeleteNoteR)

	// --------------------------------------
	// Assets
	// --------------------------------------
	r.Static("/dist", cfg.AssetsFolder)
	r.Run()
}

func initDatabase() {
	var err error
	db, err = gorm.Open(sqlite.Open(cfg.Database), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&Case{})

	db.AutoMigrate(&Event{})
	db.AutoMigrate(&Asset{})
	db.AutoMigrate(&Malware{})
	db.AutoMigrate(&Indicator{})

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Evidence{})
	db.AutoMigrate(&Task{})
	db.AutoMigrate(&Note{})
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Printf("ENV | %s = %s", key, value)
		return value
	}
	return fallback
}
