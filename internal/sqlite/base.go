package sqlite

import (
	"log"
	"os"
	"time"

	"github.com/sprungknoedl/dagobert/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

func Connect(dburl string) (*Store, error) {
	debugLog := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  false,         // Disable color
		},
	)

	var err error
	db, err := gorm.Open(sqlite.Open(dburl), &gorm.Config{Logger: debugLog})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&model.Asset{},
		&model.Case{},
		&model.Event{},
		&model.Evidence{},
		&model.Indicator{},
		&model.Malware{},
		&model.Note{},
		&model.Task{},
		&model.User{},
	)
	if err != nil {
		return nil, err
	}

	return &Store{db}, nil
}
