package model

import (
	"database/sql"
	"database/sql/driver"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var Migrations embed.FS
var DefaultUrl = "file:files/dagobert.db?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)"

type Store struct {
	RawConn *sql.DB
	DB      *gorm.DB
}

func Connect(dburl string) (*Store, error) {
	var err error
	conn, err := sql.Open("sqlite", dburl)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: conn}), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Store{RawConn: conn, DB: db}, nil
}

func (store *Store) Transaction(fn func(tx *Store) error) error {
	return store.DB.Transaction(func(tx *gorm.DB) error {
		return fn(&Store{RawConn: store.RawConn, DB: tx})
	})
}

func FromEnv(name string, defaults []string) []string {
	list := strings.Split(os.Getenv(name), ";")
	if len(list) > 1 {
		return list
	}
	return defaults
}

type Time time.Time

func (t Time) Format(layout string) string { return time.Time(t).Format(layout) }
func (t Time) IsZero() bool                { return time.Time(t).IsZero() }

func (t Time) Value() (driver.Value, error) {
	return time.Time(t).Format(time.RFC3339Nano), nil
}

func (t *Time) Scan(src interface{}) (err error) {
	switch src := src.(type) {
	case string:
		if src == "" {
			return nil
		}
		t2, err := time.Parse(time.RFC3339Nano, src)
		*t = Time(t2)
		return err
	case time.Time:
		*t = Time(src)
		return nil
	case nil:
		*t = Time(time.Time{})
		return nil
	default:
		return fmt.Errorf("incompatible type '%T' for Time", src)
	}
}

func (t *Time) UnmarshalText(text []byte) (err error) {
	t2, err := time.Parse(time.RFC3339Nano, string(text))
	*t = Time(t2)
	return err
}
