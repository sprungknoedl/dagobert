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
	kase, err := store.GetCase(obj.CaseID)
	if err != nil {
		// TODO: error logging
		return
	}

	for _, hook := range hooks {
		if hook.ConditionFn(obj) {
			log.Printf("running %s -> %s", hook.Name, hook.ModuleObj.Name())

			err := store.PushJob(model.Job{
				ID:       fp.Random(10),
				Name:     hook.ModuleObj.Name(),
				Status:   "Scheduled",
				Case:     kase,
				ObjectID: obj.ID,
				Object:   model.Object{Payload: obj},
			})
			if err != nil {
				log.Printf("error scheduling job for %s -> %s", hook.ModuleObj.Name(), err)
				return
			}
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
