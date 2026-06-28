package model

import (
	"database/sql"
	"database/sql/driver"
	"embed"
	"encoding/json"
	"errors"
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
	// the job runners poll the database concurrently; without a busy timeout
	// a second writer fails immediately with SQLITE_BUSY instead of waiting
	// for the write lock
	if strings.HasPrefix(dburl, "file:") && !strings.Contains(dburl, "busy_timeout") {
		if strings.Contains(dburl, "?") {
			dburl += "&_pragma=busy_timeout(10000)"
		} else {
			dburl += "?_pragma=busy_timeout(10000)"
		}
	}

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

// ErrForeignCase is returned by the Save* helpers when a record with the given
// primary key already exists under a different case. It is model-layer
// defense-in-depth: a GORM Save is a primary-key upsert, so without this check a
// forged id in a permitted case's URL could overwrite and hijack another case's
// record.
var ErrForeignCase = errors.New("record belongs to another case")

// assertCaseOwnership rejects a Save when a row of model with the given id is
// already owned by a case other than cid. A brand-new id (no existing row)
// passes, so creates and same-case updates are unaffected. model must be a
// pointer to a model value with id and case_id columns.
func (store *Store) assertCaseOwnership(model any, id, cid string) error {
	var n int64
	if err := store.DB.Model(model).Where("id = ? AND case_id <> ?", id, cid).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrForeignCase
	}
	return nil
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

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339Nano))
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
	s := string(text)
	if s == "" {
		*t = Time(time.Time{})
		return nil
	}
	// Try RFC3339 first (API / CSV input)
	t2, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Fall back to datetime-local format from HTML inputs (no timezone, assume UTC)
		t2, err = time.Parse("2006-01-02T15:04", s)
	}
	*t = Time(t2)
	return err
}

func (t *Time) UnmarshalJSON(text []byte) (err error) {
	str := ""
	if err = json.Unmarshal(text, &str); err != nil {
		return err
	}

	t2, err := time.Parse(time.RFC3339Nano, str)
	*t = Time(t2)
	return err
}

type Strings []string

func (o *Strings) Scan(src any) error {
	str, ok := src.(string)
	if !ok {
		return errors.New("src value cannot cast to string")
	}
	*o = strings.Split(str, ",")
	return nil
}

func (o Strings) Value() (driver.Value, error) {
	return strings.Join(o, ","), nil
}

// Custom holds the per-model custom-attribute values as a label→value map,
// JSON-serialized into a single TEXT column.
type Custom map[string]string

// Scan JSON-unmarshals the column into the map. Any unmarshal error or a
// non-string source collapses to an empty map (never an error), so the view
// layer always receives a valid map even for manually corrupted columns.
func (c *Custom) Scan(src any) error {
	str, ok := src.(string)
	if !ok || str == "" {
		*c = Custom{}
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal([]byte(str), &m); err != nil {
		*c = Custom{}
		return nil
	}
	*c = m
	return nil
}

// Value serializes the map to JSON. An empty or nil map returns "" (not "{}")
// to keep the column and CSV human-readable.
func (c Custom) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "", nil
	}
	b, err := json.Marshal(map[string]string(c))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// JSON returns the same string representation Value produces, for CSV export.
func (c Custom) JSON() string {
	v, err := c.Value()
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
