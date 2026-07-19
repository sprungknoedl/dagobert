package modules

import (
	"context"
	"testing"

	"github.com/a-h/templ"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/internal/model"
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

func (fakeModule) Name() string                                       { return "FakeTI" }
func (fakeModule) Description() string                                { return "" }
func (fakeModule) Validate() (model.Module, error)                    { return fakeModule{}, nil }
func (fakeModule) Run(context.Context, *model.Store, model.Job) error { return nil }
func (fakeModule) RenderResults() templ.Component                     { return templ.NopComponent }
func (fakeModule) RenderSettings() templ.Component                    { return templ.NopComponent }
func (fakeModule) Supports(obj any) bool {
	ind, ok := obj.(model.Indicator)
	return ok && ind.TLP != "TLP:RED"
}

func TestTriggerGating(t *testing.T) {
	store := setupWorkerDB(t)

	kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
	assert.Nil(t, store.SaveCase(kase))

	// register the fake module and a compiled OnIndicatorAdded rule
	savedModules, savedHooks := Modules, rules
	defer func() { Modules, rules = savedModules, savedHooks }()
	Modules = map[string]model.Module{"FakeTI": fakeModule{}}

	rule, err := CompileAutomationRule(model.AutomationRule{
		ID:        fp.Random(10),
		Trigger:   "OnIndicatorAdded",
		Name:      "enrich",
		Module:    "FakeTI",
		Condition: "obj.Type in ['IP','Domain','Hash','URL']",
		Enabled:   true,
	})
	assert.Nil(t, err) // condition compiles against model.Indicator
	rules.Store([]AutomationRule{rule})

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

func TestCompileAutomationRuleCaseTriggers(t *testing.T) {
	savedModules := Modules
	defer func() { Modules = savedModules }()
	Modules = map[string]model.Module{"FakeTI": fakeModule{}}

	for _, trigger := range []string{"OnCaseAdded", "OnCaseUpdated"} {
		t.Run(trigger, func(t *testing.T) {
			_, err := CompileAutomationRule(model.AutomationRule{
				ID:        fp.Random(10),
				Trigger:   trigger,
				Name:      "rule",
				Module:    "FakeTI",
				Condition: "obj.Name != ''",
				Enabled:   true,
			})
			assert.Nil(t, err) // condition compiles against model.Case
		})
	}
}

func TestTriggerOnCaseSettingsPropagation(t *testing.T) {
	store := setupWorkerDB(t)

	kase := model.Case{ID: fp.Random(10), Name: "Test Case"}
	assert.Nil(t, store.SaveCase(kase))

	savedModules, savedHooks := Modules, rules
	defer func() { Modules, rules = savedModules, savedHooks }()
	Modules = map[string]model.Module{"Webhook": webhookModule{}}

	rule, err := CompileAutomationRule(model.AutomationRule{
		ID:        fp.Random(10),
		Trigger:   "OnCaseAdded",
		Name:      "notify",
		Module:    "Webhook",
		Condition: "true",
		Enabled:   true,
		URL:       "https://example.test/hook",
	})
	assert.Nil(t, err)
	rules.Store([]AutomationRule{rule})

	TriggerOnCaseAdded(store, kase)

	jobs, err := store.GetJobs(kase.ID)
	assert.Nil(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "case.added", jobs[0].Settings["event"])
	assert.Equal(t, "https://example.test/hook", jobs[0].Settings["url"])
	assert.Equal(t, "notify", jobs[0].Settings["rule"])
}

// webhookModule is a minimal stand-in for the real Webhook module, avoiding an
// import of internal/modules/webhook (which would create an import cycle back
// into this package via automation-rules.go's rule compilation).
type webhookModule struct{}

func (webhookModule) Name() string                                       { return "Webhook" }
func (webhookModule) Description() string                                { return "" }
func (webhookModule) Validate() (model.Module, error)                    { return webhookModule{}, nil }
func (webhookModule) Run(context.Context, *model.Store, model.Job) error { return nil }
func (webhookModule) RenderSettings() templ.Component                    { return templ.NopComponent }
func (webhookModule) Supports(obj any) bool                              { return true }
