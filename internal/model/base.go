package model

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Connect(dburl string) (*Store, error) {
	var err error
	db, err := sql.Open("sqlite", dburl)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func ScanAll(rows *sql.Rows, dest any) error {
	defer rows.Close()

	destv := reflect.ValueOf(dest).Elem()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	args := make([]any, len(cols))

	for rows.Next() {
		rowp := reflect.New(destv.Type().Elem())
		rowv := rowp.Elem()

		for i := 0; i < len(cols); i++ {
			args[i] = rowv.Field(i).Addr().Interface()
		}

		if err := rows.Scan(args...); err != nil {
			return err
		}

		destv.Set(reflect.Append(destv, rowv))
	}

	return rows.Err()
}

func ScanOne(rows *sql.Rows, dest any) error {
	defer rows.Close()

	destv := reflect.ValueOf(dest).Elem()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	args := make([]any, len(cols))
	for i := 0; i < len(cols); i++ {
		args[i] = destv.Field(i).Addr().Interface()
	}

	if !rows.Next() {
		return sql.ErrNoRows
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return rows.Scan(args...)
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

const schema = `
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS auditlog (
	time     DATETIME NOT NULL,
	user      TEXT NOT NULL,
	kase     TEXT NOT NULL,
	object   TEXT NOT NULL,
	activity TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cases (
	id             TEXT NOT NULL PRIMARY KEY,
	name           TEXT NOT NULL,
	closed         BOOLEAN NOT NULL,
	classification TEXT NOT NULL,
	severity       TEXT NOT NULL,
	outcome        TEXT NOT NULL,
	summary        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assets (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	status      TEXT NOT NULL,
	type        TEXT NOT NULL,
	name        TEXT NOT NULL,
	addr        TEXT NOT NULL,
	notes       TEXT NOT NULL,

	UNIQUE (name, case_id),
	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
	id      TEXT NOT NULL PRIMARY KEY,
	case_id TEXT NOT NULL,
	time    DATETIME NOT NULL,
	type    TEXT NOT NULL,
	event   TEXT NOT NULL,
	raw     TEXT NOT NULL,
	flagged BOOLEAN NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS event_assets (
	event_id TEXT NOT NULL,
	asset_id TEXT NOT NULL,

	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS event_indicators (
	id           TEXT NOT NULL PRIMARY KEY,
	event_id     TEXT NOT NULL,
	indicator_id TEXT NOT NULL,

	FOREIGN KEY (indicator_id) REFERENCES indicators(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS evidences (
	id       TEXT NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	name     TEXT NOT NULL,
	type     TEXT NOT NULL,
	size     INTEGER NOT NULL,
	source   TEXT NOT NULL,
	notes    TEXT NOT NULL,
	hash     TEXT NOT NULL,
	location TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS reports (
	id       TEXT NOT NULL PRIMARY KEY,
	name     TEXT NOT NULL,
	notes    TEXT NOT NULL,

	UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS runs (
	evidence_id TEXT NOT NULL,
	name        TEXT NOT NULL,
	description TEXT NOT NULL,
	status      TEXT NOT NULL,
	error       TEXT NOT NULL,
	ttl         DATETIME NOT NULL,

	PRIMARY KEY (evidence_id, name),
	FOREIGN KEY (evidence_id) REFERENCES evidences(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS indicators (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	status      TEXT NOT NULL,
	type        TEXT NOT NULL,
	value       TEXT NOT NULL,
	tlp         TEXT NOT NULL,
	notes       TEXT NOT NULL,
	source      TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS keys (
	key  TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS malware (
	id       TEXT NOT NULL PRIMARY KEY,
	asset_id TEXT NOT NULL,
	case_id  TEXT NOT NULL,
	status   TEXT NOT NULL,
	name     TEXT NOT NULL,
	path     TEXT NOT NULL,
	hash     TEXT NOT NULL,
	notes    TEXT NOT NULL,

	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE ON UPDATE CASCADE
	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS notes (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	title       TEXT NOT NULL,
	category    TEXT NOT NULL,
	description TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
	id       TEXT NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	task     TEXT NOT NULL,
	done     BOOLEAN NOT NULL,
	owner    TEXT NOT NULL,
	date_due DATETIME NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS users (
	id         TEXT NOT NULL PRIMARY KEY,
	name       TEXT NOT NULL,
	upn        TEXT NOT NULL,
	email      TEXT NOT NULL,
	role       TEXT NOT NULL DEFAULT 'Read-Only',
	last_login DATETIME,

	UNIQUE (upn)
);

CREATE TABLE IF NOT EXISTS policies (
	ptype  TEXT NOT NULL,
	v0     TEXT NOT NULL DEFAULT '',
	v1     TEXT NOT NULL DEFAULT '',
	v2     TEXT NOT NULL DEFAULT '',
	v3     TEXT NOT NULL DEFAULT '',
	v4     TEXT NOT NULL DEFAULT '',
	v5     TEXT NOT NULL DEFAULT '',

	UNIQUE(ptype,v0,v1,v2,v3,v4,v5)
);

INSERT INTO policies (ptype, v0, v1, v2) VALUES 
	('p', '*', '/auth/*', '*'),
	('p', '*', '/web/*', '*'),
	('p', 'role::User', '/', 'GET'),
	('p', 'role::User', '/cases/', 'GET'),
	('p', 'role::Read-Only', '/', 'GET'),
	('p', 'role::Read-Only', '/cases/', 'GET'),
	('p', 'role::Administrator', '*', '*')
	ON CONFLICT DO NOTHING;
`
