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
	r.GET("/api/case", ListCaseR)
	r.GET("/api/case.csv", ExportCaseCsvR)
	r.GET("/api/case/:cid", GetCaseR)
	r.POST("/api/case", AddCaseR)
	r.PUT("/api/case/:cid", EditCaseR)
	r.DELETE("/api/case/:cid", DeleteCaseR)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	r.GET("/api/case/:cid/event", ListEventR)
	r.GET("/api/case/:cid/event.csv", ExportEventCsvR)
	r.GET("/api/case/:cid/event/:id", GetEventR)
	r.POST("/api/case/:cid/event", AddEventR)
	r.PUT("/api/case/:cid/event/:id", EditEventR)
	r.DELETE("/api/case/:cid/event/:id", DeleteEventR)

	// assets
	r.GET("/api/case/:cid/asset", ListAssetR)
	r.GET("/api/case/:cid/asset.csv", ExportAssetCsvR)
	r.GET("/api/case/:cid/asset/:id", GetAssetR)
	r.POST("/api/case/:cid/asset", AddAssetR)
	r.PUT("/api/case/:cid/asset/:id", EditAssetR)
	r.DELETE("/api/case/:cid/asset/:id", DeleteAssetR)

	// malware
	r.GET("/api/case/:cid/malware", ListMalwareR)
	r.GET("/api/case/:cid/malware.csv", ExportMalwareCsvR)
	r.GET("/api/case/:cid/malware/:id", GetMalwareR)
	r.POST("/api/case/:cid/malware", AddMalwareR)
	r.PUT("/api/case/:cid/malware/:id", EditMalwareR)
	r.DELETE("/api/case/:cid/malware/:id", DeleteMalwareR)

	// indicators
	r.GET("/api/case/:cid/indicator", ListIndicatorR)
	r.GET("/api/case/:cid/indicator.csv", ExportIndicatorCsvR)
	r.GET("/api/case/:cid/indicator/:id", GetIndicatorR)
	r.POST("/api/case/:cid/indicator", AddIndicatorR)
	r.PUT("/api/case/:cid/indicator/:id", EditIndicatorR)
	r.DELETE("/api/case/:cid/indicator/:id", DeleteIndicatorR)

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// users
	r.GET("/api/case/:cid/user", ListUserR)
	r.GET("/api/case/:cid/user.csv", ExportUserCsvR)
	r.GET("/api/case/:cid/user/:id", GetUserR)
	r.POST("/api/case/:cid/user", AddUserR)
	r.PUT("/api/case/:cid/user/:id", EditUserR)
	r.DELETE("/api/case/:cid/user/:id", DeleteUserR)

	// evidence
	r.GET("/api/case/:cid/evidence", ListEvidenceR)
	r.GET("/api/case/:cid/evidence.csv", ExportEvidenceCsvR)
	r.GET("/api/case/:cid/evidence/:id", GetEvidenceR)
	r.POST("/api/case/:cid/evidence", AddEvidenceR)
	r.PUT("/api/case/:cid/evidence/:id", EditEvidenceR)
	r.DELETE("/api/case/:cid/evidence/:id", DeleteEvidenceR)

	// tasks
	r.GET("/api/case/:cid/task", ListTaskR)
	r.GET("/api/case/:cid/task.csv", ExportTaskCsvR)
	r.GET("/api/case/:cid/task/:id", GetTaskR)
	r.POST("/api/case/:cid/task", AddTaskR)
	r.PUT("/api/case/:cid/task/:id", EditTaskR)
	r.DELETE("/api/case/:cid/task/:id", DeleteTaskR)

	// notes
	r.GET("/api/case/:cid/note", ListNoteR)
	r.GET("/api/case/:cid/note.csv", ExportNoteCsvR)
	r.GET("/api/case/:cid/note/:id", GetNoteR)
	r.POST("/api/case/:cid/note", AddNoteR)
	r.PUT("/api/case/:cid/note/:id", EditNoteR)
	r.DELETE("/api/case/:cid/note/:id", DeleteNoteR)

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
