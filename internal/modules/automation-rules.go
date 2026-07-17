package modules

import (
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/expr-lang/expr"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules/utils"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

type AutomationRule struct {
	model.AutomationRule
	ConditionFn func(any) bool `gorm:"-"`
	ModuleObj   model.Module   `gorm:"-"`
}

var rules = atomic.Value{}

// workerutils can not import this package (the module packages sit in
// between), so the rule trigger is wired up via a function variable.
func init() {
	utils.OnEvidenceAdded = TriggerOnEvidenceAdded
}

func LoadAutomationRules(store *model.Store) error {
	list, err := store.ListAutomationRules()
	if err != nil {
		slog.Error("error loading rule definitions", "err", err)
		return err
	}

	tmp := []AutomationRule{}
	for _, def := range list {
		if !def.Enabled {
			continue
		}

		rule, err := CompileAutomationRule(def)
		if err != nil {
			slog.Error("error compiling rule", "rule", def.Name, "condition", def.Condition, "err", err)
			continue
		}

		tmp = append(tmp, rule)
	}

	rules.Store(tmp)
	return nil
}

func TriggerOnEvidenceAdded(store *model.Store, obj model.Evidence) {
	triggerAutomationRules(store, obj, obj.CaseID, obj.ID)
}

func TriggerOnIndicatorAdded(store *model.Store, obj model.Indicator) {
	triggerAutomationRules(store, obj, obj.CaseID, obj.ID)
}

// triggerAutomationRules evaluates every enabled rule against obj and schedules a job for
// each match. Beyond the rule's expr condition it gates on the module's
// Supports(obj): a job is never scheduled for an object the module can not (or
// must not, e.g. TLP:RED) process, so the trigger and the UI stay honest.
func triggerAutomationRules(store *model.Store, obj any, caseID, objID string) {
	kase, err := store.GetCase(caseID)
	if err != nil {
		slog.Error("error loading case for automation rules", "case", caseID, "err", err)
		return
	}

	list, _ := rules.Load().([]AutomationRule)
	for _, rule := range list {
		if !rule.ConditionFn(obj) || !rule.ModuleObj.Supports(obj) {
			continue
		}

		slog.Info("running automation rule", "rule", rule.Name, "module", rule.ModuleObj.Name())
		err := store.Transaction(func(tx *model.Store) error {
			if err := tx.PushJob(model.Job{
				ID:       fp.Random(10),
				Name:     rule.ModuleObj.Name(),
				Status:   "Scheduled",
				Case:     kase,
				ObjectID: objID,
				Object:   model.Object{Payload: obj},
			}); err != nil {
				return err
			}

			// evidence access log: a rule-scheduled module run is logged at
			// schedule time (same as a manually scheduled one), with the rule
			// as the actor
			if evidence, ok := obj.(model.Evidence); ok {
				return tx.SaveEvidenceLog(evidence.CaseID, model.EvidenceLog{
					EvidenceID: evidence.ID,
					Name:       evidence.Name,
					User:       fmt.Sprintf("automation rule %q", rule.Name),
					Event:      model.EvidenceLogModuleRun,
					Details:    rule.ModuleObj.Name(),
				})
			}
			return nil
		})
		if err != nil {
			slog.Error("error scheduling job", "module", rule.ModuleObj.Name(), "err", err)
			continue
		}
	}
}

func CompileAutomationRule(rule model.AutomationRule) (AutomationRule, error) {
	compiled := AutomationRule{AutomationRule: rule}
	// search mod
	for _, mod := range Modules {
		if mod.Name() == rule.Module {
			compiled.ModuleObj = mod
			break
		}
	}
	if compiled.ModuleObj == nil {
		return AutomationRule{}, errors.New("unkown mod")
	}

	// compile condition
	var obj any
	switch compiled.Trigger {
	case "OnEvidenceAdded":
		obj = model.Evidence{}
	case "OnIndicatorAdded":
		obj = model.Indicator{}
	}

	program, err := expr.Compile(compiled.Condition,
		expr.AsBool(),
		expr.Env(map[string]any{
			"obj": obj,
		}))
	if err != nil {
		return AutomationRule{}, err
	}

	compiled.ConditionFn = func(obj any) bool {
		out, err := expr.Run(program, map[string]any{"obj": obj})
		if err != nil {
			slog.Error("error evaluating rule expression", "err", err)
			return false
		}

		return out.(bool)
	}

	// return finished compiled rule
	return compiled, nil
}
