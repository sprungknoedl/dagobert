package worker

import (
	"testing"

	"github.com/a-h/templ"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/stretchr/testify/assert"
)

func setupWorkerDB(t *testing.T) *model.Store {
	db, err := model.Connect(":memory:")
	assert.Nil(t, err)
	t.Cleanup(func() { db.RawConn.Close() })

	source, _ := iofs.New(model.Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	assert.Nil(t, m.Up())
	return db
}

// fakeModule enriches indicators only and refuses TLP:RED, mirroring the
// Supports() gate every TI module implements.
type fakeModule struct{}

func (fakeModule) Name() string                    { return "FakeTI" }
func (fakeModule) Description() string             { return "" }
func (fakeModule) Validate() (model.Module, error) { return fakeModule{}, nil }
func (fakeModule) Run(model.Job) error             { return nil }
func (fakeModule) RenderResults() templ.Component  { return templ.NopComponent }
func (fakeModule) RenderSettings() templ.Component { return templ.NopComponent }
func (fakeModule) Supports(obj any) bool {
	ind, ok := obj.(model.Indicator)
	return ok && ind.TLP != "TLP:RED"
}

func TestTriggerGating(t *testing.T) {
	store := setupWorkerDB(t)

	kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
	assert.Nil(t, store.SaveCase(kase))

	// register the fake module and a compiled OnIndicatorAdded hook
	savedModules, savedHooks := Modules, hooks
	defer func() { Modules, hooks = savedModules, savedHooks }()
	Modules = map[string]model.Module{"FakeTI": fakeModule{}}

	hook, err := CompileHook(model.Hook{
		ID:        fp.Random(10),
		Trigger:   "OnIndicatorAdded",
		Name:      "enrich",
		Module:    "FakeTI",
		Condition: "obj.Type in ['IP','Domain','Hash','URL']",
		Enabled:   true,
	})
	assert.Nil(t, err) // condition compiles against model.Indicator
	hooks = []model.Hook{hook}

	t.Run("schedules a job for a supported indicator", func(t *testing.T) {
		ind := model.Indicator{ID: fp.Random(10), CaseID: kase.ID, Type: "IP", Value: "1.2.3.4", TLP: "TLP:GREEN"}
		TriggerOnIndicatorAdded(store, ind)

		jobs, err := store.GetJobs(ind.ID)
		assert.Nil(t, err)
		assert.Len(t, jobs, 1)
	})

	t.Run("skips a TLP:RED indicator (Supports gate)", func(t *testing.T) {
		ind := model.Indicator{ID: fp.Random(10), CaseID: kase.ID, Type: "IP", Value: "5.6.7.8", TLP: "TLP:RED"}
		TriggerOnIndicatorAdded(store, ind)

		jobs, err := store.GetJobs(ind.ID)
		assert.Nil(t, err)
		assert.Len(t, jobs, 0)
	})

	t.Run("retrofitted evidence path also gates on Supports", func(t *testing.T) {
		// the indicator-only module must not grab evidence
		ev := model.Evidence{ID: fp.Random(10), CaseID: kase.ID, Type: "File", Name: "x"}
		TriggerOnEvidenceAdded(store, ev)

		jobs, err := store.GetJobs(ev.ID)
		assert.Nil(t, err)
		assert.Len(t, jobs, 0)
	})
}
