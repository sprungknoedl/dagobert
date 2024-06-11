package model

import (
	"database/sql"
	"os"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func Connect(dburl string) (*Store, error) {
	var err error
	db, err := sql.Open("sqlite3", dburl)
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

	rows.Next()
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

const schema = `
CREATE TABLE IF NOT EXISTS cases (
	id             TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	name           TEXT NOT NULL,
	closed         BOOLEAN NOT NULL,
	classification TEXT NOT NULL,
	severity       TEXT NOT NULL,
	outcome        TEXT NOT NULL,
	summary        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assets (
	id          TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	type        TEXT NOT NULL,
	name        TEXT NOT NULL,
	ip          TEXT NOT NULL,
	description TEXT NOT NULL,
	compromised BOOLEAN NOT NULL,
	analysed    BOOLEAN NOT NULL,

	UNIQUE (name, case_id),
	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
	id      TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	case_id TEXT NOT NULL,
	time    DATETIME NOT NULL,
	type    TEXT NOT NULL,
	event   TEXT NOT NULL,
	raw     TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS event_assets (
	id       TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	event_id TEXT NOT NULL,
	asset_id TEXT NOT NULL,

	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS event_indicators (
	id           TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	event_id     TEXT NOT NULL,
	indicator_id TEXT NOT NULL,

	FOREIGN KEY (indicator_id) REFERENCES indicators(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS evidences (
	id          TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	name        TEXT NOT NULL,
	type        TEXT NOT NULL,
	description TEXT NOT NULL,
	size        INTEGER NOT NULL,
	hash        TEXT NOT NULL,
	location    TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS indicators (
	id          TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	type        TEXT NOT NULL,
	value       TEXT NOT NULL,
	tlp         TEXT NOT NULL,
	description TEXT NOT NULL,
	source      TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS malware (
	id       TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	asset_id TEXT NOT NULL,
	case_id  TEXT NOT NULL,
	filename TEXT NOT NULL,
	filepath TEXT NOT NULL,
	c_date   DATETIME NOT NULL,
	m_date   DATETIME NOT NULL,
	hash     TEXT NOT NULL,
	notes    TEXT NOT NULL,

	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE ON UPDATE CASCADE
	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS notes (
	id          TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	title       TEXT NOT NULL,
	category    TEXT NOT NULL,
	description TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
	id       TEXT DEFAULT (lower(hex(randomblob(5)))) NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	task     TEXT NOT NULL,
	done     BOOLEAN NOT NULL,
	owner    TEXT NOT NULL,
	date_due DATETIME NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS users (
	id    TEXT NOT NULL PRIMARY KEY,
	name  TEXT NOT NULL,
	upn   TEXT NOT NULL,
	email TEXT NOT NULL,

	UNIQUE (upn)
);
`
