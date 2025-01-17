package model

import (
	"database/sql"
)

var TaskTypes = FromEnv("VALUES_TASK_TYPES", []string{"Information request", "Analysis", "Deliverable", "Checkpoint", "Other"})

type Task struct {
	ID      string
	Type    string
	Task    string
	Done    bool
	Owner   string
	DateDue Time
	CaseID  string
}

func (store *Store) ListTasks(cid string) ([]Task, error) {
	query := `
	SELECT id, type, task, done, owner, date_due, case_id
	FROM tasks
	WHERE case_id = :cid
	ORDER BY date_due ASC`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid))
	if err != nil {
		return nil, err
	}

	var list []Task
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetTask(cid string, id string) (Task, error) {
	query := `
	SELECT id, type, task, done, owner, date_due, case_id
	FROM tasks
	WHERE case_id = :cid AND id = :id
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("cid", cid),
		sql.Named("id", id))
	if err != nil {
		return Task{}, err
	}

	var obj Task
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveTask(cid string, obj Task) error {
	query := `
	INSERT INTO tasks (id, type, task, done, owner, date_due, case_id)
	VALUES (iif(:id != '', :id, lower(hex(randomblob(5)))), :type, :task, :done, :owner, :datedue, :cid)
	ON CONFLICT (id)
		DO UPDATE SET type=:type, task=:task, done=:done, owner=:owner, date_due=:datedue
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("cid", cid),
		sql.Named("id", obj.ID),
		sql.Named("type", obj.Type),
		sql.Named("task", obj.Task),
		sql.Named("done", obj.Done),
		sql.Named("owner", obj.Owner),
		sql.Named("datedue", obj.DateDue))
	return err
}

func (store *Store) DeleteTask(cid string, id string) error {
	query := `
	DELETE FROM tasks
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
