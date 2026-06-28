package worker

import (
	"errors"
	"log"

	"github.com/expr-lang/expr"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker/workerutils"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

var hooks = []model.Hook{}

// workerutils can not import this package (the module packages sit in
// between), so the hook trigger is wired up via a function variable.
func init() {
	workerutils.OnEvidenceAdded = TriggerOnEvidenceAdded
}

func LoadHooks(store *model.Store) error {
	list, err := store.ListHooks()
	if err != nil {
		log.Printf("error loading hook definitions: %v", err)
		return err
	}

	// LoadHooks is re-invoked on every SaveHook/DeleteHook. Reset hook
	// list to avoid accumulation of duplicates.
	hooks = []model.Hook{}
	for _, def := range list {
		if !def.Enabled {
			continue
		}

		hook, err := CompileHook(def)
		if err != nil {
			log.Printf("error compiling hook %q (%s): %v", def.Name, def.Condition, err)
			continue
		}

		hooks = append(hooks, hook)
	}

	return nil
}

func TriggerOnEvidenceAdded(store *model.Store, obj model.Evidence) {
	triggerHooks(store, obj, obj.CaseID, obj.ID)
}

func TriggerOnIndicatorAdded(store *model.Store, obj model.Indicator) {
	triggerHooks(store, obj, obj.CaseID, obj.ID)
}

// triggerHooks evaluates every enabled hook against obj and schedules a job for
// each match. Beyond the hook's expr condition it gates on the module's
// Supports(obj): a job is never scheduled for an object the module can not (or
// must not, e.g. TLP:RED) process, so the trigger and the UI stay honest.
func triggerHooks(store *model.Store, obj any, caseID, objID string) {
	kase, err := store.GetCase(caseID)
	if err != nil {
		// TODO: error logging
		return
	}

	for _, hook := range hooks {
		if !hook.ConditionFn(obj) || !hook.ModuleObj.Supports(obj) {
			continue
		}

		log.Printf("running %s -> %s", hook.Name, hook.ModuleObj.Name())
		err := store.PushJob(model.Job{
			ID:       fp.Random(10),
			Name:     hook.ModuleObj.Name(),
			Status:   "Scheduled",
			Case:     kase,
			ObjectID: objID,
			Object:   model.Object{Payload: obj},
		})
		if err != nil {
			log.Printf("error scheduling job for %s -> %s", hook.ModuleObj.Name(), err)
			return
		}
	}
}

func CompileHook(hook model.Hook) (model.Hook, error) {
	// search mod
	for _, mod := range Modules {
		if mod.Name() == hook.Module {
			hook.ModuleObj = mod
			break
		}
	}
	if hook.ModuleObj == nil {
		return model.Hook{}, errors.New("unkown mod")
	}

	// compile condition
	var obj any
	switch hook.Trigger {
	case "OnEvidenceAdded":
		obj = model.Evidence{}
	case "OnIndicatorAdded":
		obj = model.Indicator{}
	}

	program, err := expr.Compile(hook.Condition,
		expr.AsBool(),
		expr.Env(map[string]any{
			"obj": obj,
		}))
	if err != nil {
		return model.Hook{}, err
	}

	hook.ConditionFn = func(obj any) bool {
		out, err := expr.Run(program, map[string]any{"obj": obj})
		if err != nil {
			log.Printf("error evaluating hook expression: %v", err)
			return false
		}

		return out.(bool)
	}

	// return finished hook
	return hook, nil
}
