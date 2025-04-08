package model

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/fp"
)

var ServerToken = fp.Random(20)

var JobStatus = []string{
	"Scheduled",
	"Running",
	"Failed",
	"Success",
}

var HookTrigger = []string{
	"OnEvidenceAdded",
}

type Job struct {
	ID          string
	CaseID      string
	EvidenceID  string
	Name        string
	Status      string
	Error       string
	ServerToken string
	WorkerToken string

	Description string
}

type Hook struct {
	ID        string
	Trigger   string
	Name      string
	Mod       string
	Condition string
	Enabled   bool
}

func (store *Store) GetJobs(eid string) ([]Job, error) {
	query := `
	SELECT id, case_id, evidence_id, name, status, error, server_token, worker_token
	FROM jobs
	WHERE evidence_id = :eid`

	rows, err := store.DB.Query(query,
		sql.Named("eid", eid))
	if err != nil {
		return nil, err
	}

	list := []Job{}
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetRunningJobs() ([]Job, error) {
	query := `
	SELECT id, case_id, evidence_id, name, status, error, server_token, worker_token
	FROM jobs
	WHERE status = :status`

	rows, err := store.DB.Query(query,
		sql.Named("status", "Running"))
	if err != nil {
		return nil, err
	}

	list := []Job{}
	err = ScanAll(rows, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (store *Store) SaveJob(obj Job) error {
	query := `
	REPLACE INTO jobs (id, case_id, evidence_id, name, status, error, server_token, worker_token)
	VALUES (:id, :case_id, :evidence_id, :name, :status, :error, :stoken, :wtoken)`

	obj.ServerToken = ServerToken
	_, err := store.DB.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("case_id", obj.CaseID),
		sql.Named("evidence_id", obj.EvidenceID),
		sql.Named("name", obj.Name),
		sql.Named("status", obj.Status),
		sql.Named("error", obj.Error),
		sql.Named("stoken", obj.ServerToken),
		sql.Named("wtoken", obj.WorkerToken))
	return err
}

func (store *Store) PushJob(obj Job) error { return store.SaveJob(obj) }
func (store *Store) PopJob(workerid string, modules []string) (Job, Case, Evidence, error) {
	// slices are not supported as parameterized arguments in database/sql and sqlite.
	// we have to use a workaround to pass the list of modules as a single argument.
	re := regexp.MustCompile("[a-zA-Z0-9_ ]+")
	for _, m := range modules {
		if !re.MatchString(m) {
			return Job{}, Case{}, Evidence{}, fmt.Errorf("invalid module name: %q", m)
		}
	}

	modules = fp.Apply(modules, func(s string) string { return "'" + s + "'" })

	query := `
	UPDATE jobs
	SET status = :status_after, worker_token = :wtoken
	WHERE rowid = (
		SELECT min(rowid)
		FROM jobs
		WHERE status = :status_before 
		AND name IN (` + strings.Join(modules, ", ") + `) )
	RETURNING id, case_id, evidence_id, name, status, error, server_token, worker_token;
	`

	rows, err := store.DB.Query(query,
		sql.Named("status_before", "Scheduled"),
		sql.Named("status_after", "Running"),
		sql.Named("wtoken", workerid))
	if err != nil {
		return Job{}, Case{}, Evidence{}, err
	}

	obj := Job{}
	err = ScanOne(rows, &obj)
	if err != nil {
		return Job{}, Case{}, Evidence{}, err
	}

	// fetch objects
	evidence, err1 := store.GetEvidence(obj.CaseID, obj.EvidenceID)
	kase, err2 := store.GetCase(obj.CaseID)
	if err := errors.Join(err1, err2); err != nil {
		return Job{}, Case{}, Evidence{}, err
	}

	return obj, kase, evidence, err
}

func (store *Store) AckJob(id string, status string, errmsg string) error {
	query := `
	UPDATE jobs
	SET status = :status, error = :error
	WHERE id == :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id),
		sql.Named("status", status),
		sql.Named("error", errmsg))
	return err
}

func (store *Store) RescheduleWorkerJobs(workerToken string) error {
	query := `
	UPDATE jobs
	SET status = :status_after, worker_token = ''
	WHERE worker_token == :wtoken
	AND status = :status_before`

	_, err := store.DB.Exec(query,
		sql.Named("status_before", "Running"),
		sql.Named("status_after", "Scheduled"),
		sql.Named("wtoken", workerToken))
	return err
}

func (store *Store) RescheduleStaleJobs() error {
	query := `
	UPDATE jobs
	SET status = :status_after, server_token = :stoken, worker_token = ''
	WHERE server_token != :stoken
	AND status = :status_before`

	_, err := store.DB.Exec(query,
		sql.Named("status_before", "Running"),
		sql.Named("status_after", "Scheduled"),
		sql.Named("stoken", ServerToken))
	return err
}

func (store *Store) ListHooks() ([]Hook, error) {
	query := `
	SELECT id, trigger, name, mod, condition, enabled
	FROM hooks`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	list := []Hook{}
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetHook(id string) (Hook, error) {
	query := `
	SELECT id, trigger, name, mod, condition, enabled
	FROM hooks
	WHERE id = :id`

	rows, err := store.DB.Query(query,
		sql.Named("id", id))
	if err != nil {
		return Hook{}, err
	}

	obj := Hook{}
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveHook(obj Hook) error {
	query := `
	INSERT INTO hooks (id, trigger, name, mod, condition, enabled)
	VALUES (:id, :trigger, :name, :mod, :condition, :enabled)
	ON CONFLICT (id)
		DO UPDATE SET trigger=:trigger, name=:name, mod=:mod, condition=:condition, enabled=:enabled
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("trigger", obj.Trigger),
		sql.Named("name", obj.Name),
		sql.Named("mod", obj.Mod),
		sql.Named("condition", obj.Condition),
		sql.Named("enabled", obj.Enabled))
	return err
}

func (store *Store) DeleteHook(id string) error {
	query := `
	DELETE FROM hooks
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
