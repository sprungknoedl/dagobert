package model

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/a-h/templ"
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

type Module interface {
	Name() string
	Description() string
	Validate() (Module, error)
	Supports(any) bool
	Run(Job) error
	RenderResults() templ.Component
	RenderSettings() templ.Component
}

type Job struct {
	ID          string
	Name        string
	Status      string
	Error       string
	Results     map[string]string `gorm:"serializer:json"`
	Settings    map[string]string `gorm:"serializer:json"`
	ServerToken string
	WorkerToken string

	CaseID   string
	Case     Case
	ObjectID string
	Object   Object `gorm:"type:bytes"`

	Module Module          `gorm:"-"` // used in handler/jobs.go to pass module information (like name and description) to ui
	Ctx    context.Context `gorm:"-"` // used in worker/runner.go to pass request context through go channel to job runners
}

type Hook struct {
	ID        string
	Trigger   string
	Name      string
	Module    string
	Condition string
	Enabled   bool

	ConditionFn func(any) bool `gorm:"-"`
	ModuleObj   Module         `gorm:"-"`
}

type envelope struct {
	Kind string
	Data []byte
}

type Object struct {
	Payload any
}

func (o *Object) Scan(src any) error {
	if src == nil {
		o.Payload = nil
		return nil
	}

	// GORM usually gives us []byte or string for BLOBs
	data, ok := src.([]byte)
	if !ok {
		return errors.New("gorm object: database did not return []byte")
	}

	return o.UnmarshalJSON(data)
}

func (o *Object) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Step A: Decode the outer envelope
	var env envelope
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&env); err != nil {
		return err
	}

	// Step B: Switch on 'Kind' to create the correct concrete type
	// This avoids the "local interface" panic because we decode into concrete structs.
	dec = json.NewDecoder(bytes.NewReader(env.Data))

	switch env.Kind {
	case "":
		// no type given -> empty payload
	case "evidence":
		var dst Evidence
		if err := dec.Decode(&dst); err != nil {
			return err
		}
		o.Payload = dst
	case "indicator":
		var dst Indicator
		if err := dec.Decode(&dst); err != nil {
			return err
		}
		o.Payload = dst
	case "malware":
		var dst Malware
		if err := dec.Decode(&dst); err != nil {
			return err
		}
		o.Payload = dst
	default:
		return fmt.Errorf("gorm object: unknown kind '%s'", env.Kind)
	}

	return nil
}

func (o Object) Value() (driver.Value, error) {
	return o.MarshalJSON()
}

func (o Object) MarshalJSON() ([]byte, error) {
	if o.Payload == nil {
		return json.Marshal(nil)
	}

	// Step A: Identify the type and encode the specific payload into bytes
	var kind string
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	switch v := o.Payload.(type) {
	case Evidence:
		kind = "evidence"
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
	case Indicator:
		kind = "indicator"
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
	case Malware:
		kind = "malware"
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("gorm object: unsupported type")
	}

	// Step B: Wrap it in the envelope
	env := envelope{
		Kind: kind,
		Data: buf.Bytes(),
	}

	// Step C: Encode the envelope itself into a byte slice for DB
	var val bytes.Buffer
	enc = json.NewEncoder(&val)
	if err := enc.Encode(env); err != nil {
		return nil, err
	}

	return val.Bytes(), nil
}

func (store *Store) GetJobs(eid string) ([]Job, error) {
	list := []Job{}
	tx := store.DB.
		Where("object_id = ?", eid).
		Preload("Case", nil).
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetRunningJobs() ([]Job, error) {
	list := []Job{}
	tx := store.DB.
		Where("status = ?", "Running").
		Preload("Case", nil).
		Find(&list)
	return list, tx.Error
}

func (store *Store) SaveJob(obj Job) error {
	return store.DB.Save(&obj).Error
}

func (store *Store) PushJob(obj Job) error { return store.SaveJob(obj) }
func (store *Store) PopJob(workerid string, modules []string) (Job, error) {
	// slices are not supported as parameterized arguments in database/sql and sqlite.
	// we have to use a workaround to pass the list of modules as a single argument.
	re := regexp.MustCompile("^[()a-zA-Z0-9_ ]+$")
	for _, m := range modules {
		if !re.MatchString(m) {
			return Job{}, fmt.Errorf("invalid module name: %q", m)
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
		return Job{}, err
	}
	if obj.ID == "" {
		return Job{}, gorm.ErrRecordNotFound
	}

	return obj, err
}

func (store *Store) AckJob(job Job) error {
	results, err := json.Marshal(job.Results)
	if err != nil {
		return err
	}

	return store.DB.Model(job).
		Where("id = ?", job.ID).
		Updates(map[string]any{"status": job.Status, "error": job.Error, "results": string(results)}).
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
