package model

import (
	"database/sql"
	"time"
)

type Auditlog struct {
	Time     Time
	User     string
	Kase     string // typo required to not conflict with SQL keyword
	Object   string
	Activity string
}

func (store *Store) ListAuditlog() ([]Auditlog, error) {
	query := `
	SELECT time, user, kase, object, activity
	FROM auditlog
	ORDER BY time ASC
	LIMIT 1000`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	list := []Auditlog{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (store *Store) ListAuditlogForObject(object string) ([]Auditlog, error) {
	query := `
	SELECT time, user, kase, object, activity
	FROM auditlog
	WHERE object = :object
	ORDER BY time ASC
	LIMIT 1000`

	rows, err := store.DB.Query(query, sql.Named("object", object))
	if err != nil {
		return nil, err
	}

	list := []Auditlog{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (store *Store) ListAuditlogForUser(user string) ([]Auditlog, error) {
	query := `
	SELECT time, user, kase, object, activity
	FROM auditlog
	WHERE user = :user
	ORDER BY time ASC
	LIMIT 1000`

	rows, err := store.DB.Query(query, sql.Named("user", user))
	if err != nil {
		return nil, err
	}

	list := []Auditlog{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (store *Store) SaveAuditlog(user User, kase Case, obj string, activity string) error {
	act := Auditlog{
		Time:     Time(time.Now()),
		User:     user.String(),
		Kase:     kase.String(),
		Object:   obj,
		Activity: activity,
	}

	query := `
	INSERT INTO auditlog (time, user, kase, object, activity)
	VALUES (:time, :user, :kase, :object, :activity)`

	_, err := store.DB.Exec(query,
		sql.Named("time", act.Time),
		sql.Named("user", act.User),
		sql.Named("kase", act.Kase),
		sql.Named("object", act.Object),
		sql.Named("activity", act.Activity))

	return err
}
