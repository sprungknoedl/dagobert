package handler

import (
	"errors"
	"log"

	"github.com/expr-lang/expr"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

var hooks = []Hook{}

type Hook struct {
	Name      string
	Condition func(model.Evidence) bool
	Mod       *worker.Module
}

func LoadHooks(store *model.Store) error {
	list, err := store.ListHooks()
	if err != nil {
		log.Printf("error loading hook definitions: %v", err)
		return err
	}

	hooks = []Hook{}
	for _, def := range list {
		if !def.Enabled {
			continue
		}

		hook, err := compile(def)
		if err != nil {
			log.Printf("error compiling hook %q (%s): %v", def.Name, def.Condition, err)
			continue
		}

		hooks = append(hooks, hook)
	}

	return nil
}

func TriggerOnEvidenceAdded(store *model.Store, obj model.Evidence) {
	for _, hook := range hooks {
		if hook.Condition(obj) {
			log.Printf("running %s -> %s", hook.Name, hook.Mod.Name)
			// go Run(store, hook.Mod.Name, obj)

			err := store.PushJob(model.Job{
				ID:         fp.Random(10),
				CaseID:     obj.CaseID,
				EvidenceID: obj.ID,
				Name:       hook.Mod.Name,
				Status:     "Scheduled",
			})
			if err != nil {
				log.Printf("error scheduling job for %s -> %s", hook.Mod.Name, err)
				return
			}
		}
	}
}

func compile(def model.Hook) (Hook, error) {
	hook := Hook{Name: def.Name}

	// search mod
	for _, mod := range worker.List {
		if mod.Name == def.Mod {
			hook.Mod = &mod
			break
		}
	}
	if hook.Mod == nil {
		return hook, errors.New("unkown mod")
	}

	// compile condition
	program, err := expr.Compile(def.Condition,
		expr.AsBool(),
		expr.Env(map[string]any{
			"evidence": model.Evidence{},
		}))
	if err != nil {
		return hook, err
	}
	hook.Condition = func(e model.Evidence) bool {
		out, err := expr.Run(program, map[string]any{
			"evidence": e,
		})
		if err != nil {
			log.Printf("error evaluating hook expression: %v", err)
			return false
		}

		log.Printf("expr result: %v", out)
		return out.(bool)
	}

	// return finished hook
	return hook, nil
}
