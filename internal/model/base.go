package model

import (
	"database/sql"
	"database/sql/driver"
	"embed"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var Migrations embed.FS

type Store struct {
	DB *sql.DB
}

func Connect(dburl string) (*Store, error) {
	var err error
	db, err := sql.Open("sqlite", dburl)
	if err != nil {
		return nil, err
	}

	return &Store{DB: db}, nil
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
