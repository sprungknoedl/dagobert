package handler

import (
	"path/filepath"
	"regexp"
	"slices"

	"github.com/expr-lang/expr"
	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

var regexIP = regexp.MustCompile(`^$|^(?:\d{1,3}\.){3}\d{1,3}$`)

func ValidateAsset(dto model.Asset) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !slices.Contains(model.AssetStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.AssetTypes, dto.Type)},
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Addr", Message: "Invalid format, expected e.g. '203.0.113.1'.", Invalid: !regexIP.MatchString(dto.Addr)},
	})
}

func ValidateCase(dto model.Case) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Severity", Message: "Invalid value.", Invalid: !slices.Contains(model.CaseSeverities, dto.Severity)},
		{Name: "Outcome", Message: "Invalid value.", Invalid: !slices.Contains(model.CaseOutcomes, dto.Outcome)},
		{Name: "SketchID", Message: "Invalid value. Must be positive.", Invalid: dto.SketchID < 0},
	})
}

func ValidateEvent(dto model.Event) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Time", Missing: dto.Time.IsZero()},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.EventTypes, dto.Type)},
		{Name: "Event", Missing: dto.Event == ""},
	})
}

func ValidateEvidence(dto model.Evidence) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Type", Missing: dto.Type == ""},
	})
}

func ValidateIndicator(dto model.Indicator) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Value", Missing: dto.Value == ""},
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !slices.Contains(model.IndicatorStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.IndicatorTypes, dto.Type)},
		{Name: "TLP", Message: "Invalid value.", Missing: dto.Type == "", Invalid: !slices.Contains(model.IndicatorTLPs, dto.TLP)},
	})
}

func ValidateKey(dto model.Key) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.KeyTypes, dto.Type)},
	})
}

func ValidateMalware(dto model.Malware) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Path", Missing: dto.Path == ""},
		{Name: "Source", Missing: dto.Asset.ID == ""},
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !slices.Contains(model.MalwareStatus, dto.Status)},
	})
}

func ValidateNote(dto model.Note) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Category", Missing: dto.Category == ""},
		{Name: "Title", Missing: dto.Title == ""},
	})
}

func ValidateTask(dto model.Task) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Task", Missing: dto.Task == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.TaskTypes, dto.Type)},
		{Name: "DateDue", Message: "Invalid format, expected e.g. '2006-01-02'."},
	})
}

func ValidateReport(dto model.Report) valid.Result {
	return valid.Check([]valid.Condition{
		{
			Name: "Name", Missing: dto.Name == "",
			Invalid: !slices.Contains([]string{".odt", ".ods", ".odp", ".docx"}, filepath.Ext(dto.Name)),
			Message: "Unsupported file type.",
		},
	})
}

func ValidateUser(dto model.User) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "ID", Missing: dto.ID == ""},
		{Name: "Role", Message: "Invalid role", Invalid: !slices.Contains(model.UserRoles, dto.Role)},
	})
}

func ValidateHook(dto model.Hook) valid.Result {
	mods := fp.Apply(worker.List, func(m worker.Module) string { return m.Name })

	// compile condition
	msg := ""
	_, err := expr.Compile(dto.Condition,
		expr.AsBool(),
		expr.Env(map[string]any{
			"evidence": model.Evidence{},
		}))
	if err != nil {
		msg = err.Error()
	}

	return valid.Check([]valid.Condition{
		{Name: "ID", Missing: dto.ID == ""},
		{Name: "Trigger", Message: "Invalid trigger", Invalid: !slices.Contains(model.HookTrigger, dto.Trigger)},
		{Name: "Mod", Message: "Invalid mod", Invalid: !slices.Contains(mods, dto.Mod)},
		{Name: "Condition", Message: msg, Invalid: err != nil},
	})
}
