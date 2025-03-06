package mod

import (
	"errors"
	"log"

	"github.com/expr-lang/expr"
	"github.com/sprungknoedl/dagobert/internal/model"
)

var hooks = []Hook{}

type Hook struct {
	Name      string
	Condition func(model.Evidence) bool
	Mod       *Mod
}

func Compile(def model.Hook) (Hook, error) {
	hook := Hook{Name: def.Name}

	// search mod
	for _, mod := range list {
		if mod.Name == def.Mod {
			hook.Mod = &mod
			break
		}
	}
	if hook.Mod == nil {
		return hook, errors.New("unkown mod")
	}

	// compile condition
	program, err := expr.Compile(`evidence.Name endsWith ".evtx"`,
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

func InitializeHooks(store *model.Store) error {
	mu.Lock()
	defer mu.Unlock()

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

		hook, err := Compile(def)
		if err != nil {
			log.Printf("error compiling hook %q (%s): %v", def.Name, def.Condition, err)
			continue
		}

		hooks = append(hooks, hook)
	}

	return nil
}

func TriggerOnEvidenceAdded(store *model.Store, obj model.Evidence) {
	mu.Lock()
	defer mu.Unlock()

	for _, hook := range hooks {
		if hook.Condition(obj) {
			log.Printf("running %s -> %s", hook.Name, hook.Mod.Name)
			go Run(store, hook.Mod.Name, obj)
			// go hook.Mod.Run(store, obj)
		}
	}
}
