package model

import (
	"database/sql"
	"time"
)

var TaskTypes = FromEnv("VALUES_TASK_TYPES", []string{"Information request", "Analysis", "Deliverable", "Checkpoint", "Other"})

type Task struct {
	ID      string
	Type    string
	Task    string
	Done    bool
	Owner   string
	DateDue time.Time
	CaseID  string
}

func (store *Store) FindTasks(cid string, search string, sort string) ([]Task, error) {
	query := `
	SELECT id, type, task, done, owner, date_due, case_id
	FROM tasks
	WHERE case_id = :cid AND (
		instr(type, :search) > 0 OR
		instr(task, :search) > 0 OR
		instr(owner, :search) > 0)
	ORDER BY
		CASE WHEN :sort = 'type'   THEN type END ASC,
		CASE WHEN :sort = '-type'  THEN type END DESC,
		CASE WHEN :sort = 'task'   THEN task END ASC,
		CASE WHEN :sort = '-task'  THEN task END DESC,
		CASE WHEN :sort = 'owner'  THEN owner END ASC,
		CASE WHEN :sort = '-owner' THEN owner END DESC,
		CASE WHEN :sort = '-due'   THEN date_due END DESC,
		date_due ASC`

	rows, err := store.db.Query(query,
		sql.Named("cid", cid),
		sql.Named("search", search),
		sql.Named("sort", sort))
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

	rows, err := store.db.Query(query,
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

	_, err := store.db.Exec(query,
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

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
