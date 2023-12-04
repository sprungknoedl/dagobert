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
		Scopes:       []string{"openid"},
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
	r.GET("/api/case/:id", GetCaseR)
	r.POST("/api/case", AddCaseR)
	r.PUT("/api/case/:id", EditCaseR)
	r.DELETE("/api/case/:id", DeleteCaseR)

	// --------------------------------------
	// Investigation
	// --------------------------------------
	// events
	r.GET("/api/event", ListEventR)
	r.GET("/api/event/:id", GetEventR)
	r.POST("/api/event", AddEventR)
	r.PUT("/api/event/:id", EditEventR)
	r.DELETE("/api/event/:id", DeleteEventR)

	// assets
	r.GET("/api/asset", ListAssetR)
	r.GET("/api/asset/:id", GetAssetR)
	r.POST("/api/asset", AddAssetR)
	r.PUT("/api/asset/:id", EditAssetR)
	r.DELETE("/api/asset/:id", DeleteAssetR)

	// malware
	r.GET("/api/malware", ListMalwareR)
	r.GET("/api/malware/:id", GetMalwareR)
	r.POST("/api/malware", AddMalwareR)
	r.PUT("/api/malware/:id", EditMalwareR)
	r.DELETE("/api/malware/:id", DeleteMalwareR)

	// indicators
	r.GET("/api/indicator", ListIndicatorR)
	r.GET("/api/indicator/:id", GetIndicatorR)
	r.POST("/api/indicator", AddIndicatorR)
	r.PUT("/api/indicator/:id", EditIndicatorR)
	r.DELETE("/api/indicator/:id", DeleteIndicatorR)

	// --------------------------------------
	// Case Management
	// --------------------------------------
	// users
	r.GET("/api/user", ListUserR)
	r.GET("/api/user/:id", GetUserR)
	r.POST("/api/user", AddUserR)
	r.PUT("/api/user/:id", EditUserR)
	r.DELETE("/api/user/:id", DeleteUserR)

	// evidence
	r.GET("/api/evidence", ListEvidenceR)
	r.GET("/api/evidence/:id", GetEvidenceR)
	r.POST("/api/evidence", AddEvidenceR)
	r.PUT("/api/evidence/:id", EditEvidenceR)
	r.DELETE("/api/evidence/:id", DeleteEvidenceR)

	// tasks
	r.GET("/api/task", ListTaskR)
	r.GET("/api/task/:id", GetTaskR)
	r.POST("/api/task", AddTaskR)
	r.PUT("/api/task/:id", EditTaskR)
	r.DELETE("/api/task/:id", DeleteTaskR)

	// notes
	r.GET("/api/note", ListNoteR)
	r.GET("/api/note/:id", GetNoteR)
	r.POST("/api/note", AddNoteR)
	r.PUT("/api/note/:id", EditNoteR)
	r.DELETE("/api/note/:id", DeleteNoteR)

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
