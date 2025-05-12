package handler

import (
	"path/filepath"
	"slices"

	"github.com/expr-lang/expr"
	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func InEnum(enum []model.EnumItem, item string) bool {
	return slices.Contains(
		fp.Apply(enum, func(e model.EnumItem) string { return e.Name }),
		item)
}

func ValidateAsset(dto model.Asset, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !InEnum(enums.AssetStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !InEnum(enums.AssetTypes, dto.Type)},
		{Name: "Name", Missing: dto.Name == ""},
	})
}

func ValidateCase(dto model.Case, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Severity", Message: "Invalid value.", Invalid: !InEnum(enums.CaseSeverities, dto.Severity)},
		{Name: "Outcome", Message: "Invalid value.", Invalid: !InEnum(enums.CaseOutcomes, dto.Outcome)},
		{Name: "SketchID", Message: "Invalid value. Must be positive.", Invalid: dto.SketchID < 0},
	})
}

func ValidateEvent(dto model.Event, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Time", Missing: dto.Time.IsZero()},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !InEnum(enums.EventTypes, dto.Type)},
		{Name: "Event", Missing: dto.Event == ""},
	})
}

func ValidateEvidence(dto model.Evidence, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Type", Missing: dto.Type == ""},
	})
}

func ValidateIndicator(dto model.Indicator, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Value", Missing: dto.Value == ""},
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !InEnum(enums.IndicatorStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !InEnum(enums.IndicatorTypes, dto.Type)},
		{Name: "TLP", Message: "Invalid value.", Missing: dto.Type == "", Invalid: !InEnum(enums.IndicatorTLPs, dto.TLP)},
	})
}

func ValidateKey(dto model.Key, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !InEnum(enums.KeyTypes, dto.Type)},
	})
}

func ValidateMalware(dto model.Malware, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Path", Missing: dto.Path == ""},
		{Name: "Source", Missing: dto.Asset.ID == ""},
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !InEnum(enums.MalwareStatus, dto.Status)},
	})
}

func ValidateNote(dto model.Note, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Category", Missing: dto.Category == ""},
		{Name: "Title", Missing: dto.Title == ""},
	})
}

func ValidateTask(dto model.Task, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Task", Missing: dto.Task == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !InEnum(enums.TaskTypes, dto.Type)},
		{Name: "DateDue", Message: "Invalid format, expected e.g. '2006-01-02'."},
	})
}

func ValidateReport(dto model.Report, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{
			Name: "Name", Missing: dto.Name == "",
			Invalid: !slices.Contains([]string{".odt", ".ods", ".odp", ".docx"}, filepath.Ext(dto.Name)),
			Message: "Unsupported file type.",
		},
	})
}

func ValidateUser(dto model.User, enums model.Enums) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "ID", Missing: dto.ID == ""},
		{Name: "Role", Message: "Invalid role", Invalid: !slices.Contains(model.UserRoles, dto.Role)},
	})
}

func ValidateHook(dto model.Hook, enums model.Enums) valid.Result {
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
		{Name: "Trigger", Message: "Invalid trigger", Invalid: !slices.Contains(enums.HookTrigger, dto.Trigger)},
		{Name: "Mod", Message: "Invalid mod", Invalid: !slices.Contains(mods, dto.Mod)},
		{Name: "Condition", Message: msg, Invalid: err != nil},
	})
}
