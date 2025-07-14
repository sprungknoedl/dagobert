package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sprungknoedl/dagobert/pkg/fp"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ServerToken = fp.Random(20)

var JobStatus = []string{
	"Scheduled",
	"Running",
	"Failed",
	"Success",
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

	Description string `gorm:"-"`
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
	list := []Job{}
	tx := store.DB.
		Where("evidence_id = ?", eid).
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetRunningJobs() ([]Job, error) {
	list := []Job{}
	tx := store.DB.
		Where("status = ?", "Running").
		Find(&list)
	return list, tx.Error
}

func (store *Store) SaveJob(obj Job) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) PushJob(obj Job) error { return store.SaveJob(obj) }
func (store *Store) PopJob(workerid string, modules []string) (Job, Case, Evidence, error) {
	// slices are not supported as parameterized arguments in database/sql and sqlite.
	// we have to use a workaround to pass the list of modules as a single argument.
	re := regexp.MustCompile("^[()a-zA-Z0-9_ ]+$")
	for _, m := range modules {
		if !re.MatchString(m) {
			return Job{}, Case{}, Evidence{}, fmt.Errorf("invalid module name: %q", m)
		}
	}

	modules = fp.Apply(modules, func(s string) string { return "'" + s + "'" })
	rowid := store.DB.Model(Job{}).Select("min(rowid)").Where("status = ? and name in ("+strings.Join(modules, ", ")+")", "Scheduled")

	obj := Job{}
	err := store.DB.Model(&obj).
		Clauses(clause.Returning{}).
		Where("rowid = (?)", rowid).
		Updates(map[string]any{"status": "Running", "worker_token": workerid}).
		Error
	if err != nil {
		return Job{}, Case{}, Evidence{}, err
	}
	if obj.ID == "" {
		return Job{}, Case{}, Evidence{}, gorm.ErrRecordNotFound
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
	return store.DB.Model(&Job{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "error": errmsg}).
		Error
}

func (store *Store) RescheduleWorkerJobs(workerToken string) error {
	return store.DB.Model(&Job{}).
		Where("worker_token = ? and status = ?", workerToken, "Running").
		Updates(map[string]any{"status": "Scheduled", "worker_token": ""}).
		Error
}

func (store *Store) RescheduleStaleJobs() error {
	return store.DB.Model(&Job{}).
		Where("server_token = ? and status = ?", ServerToken, "Running").
		Updates(map[string]any{"status": "Scheduled", "server_token": ServerToken, "worker_token": ""}).
		Error
}

func (store *Store) ListHooks() ([]Hook, error) {
	list := []Hook{}
	tx := store.DB.Find(&list)
	return list, tx.Error
}

func (store *Store) GetHook(id string) (Hook, error) {
	obj := Hook{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveHook(obj Hook) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) DeleteHook(id string) error {
	return store.DB.Delete(Case{}, "id = ?", id).Error
}
