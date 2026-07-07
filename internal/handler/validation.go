package handler

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

// reAlnum matches a non-empty run of ASCII letters and digits only. Values used
// as a path component (a malware hash, an imported case id) must satisfy it: an
// alphanumeric string cannot contain "/", "\" or "." and so cannot traverse out
// of its target directory.
var reAlnum = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// isFlatName reports whether name is a single path element that stays inside its
// target directory — i.e. safe to join into a filesystem path. It rejects "."
// and ".." as well as any backslash, a path separator on Windows that
// filepath.Base does not collapse on Linux/macOS.
func isFlatName(name string) bool {
	return name == filepath.Base(name) && name != "." && name != ".." && !strings.ContainsRune(name, '\\')
}

func inValueList(list []model.ValueListItem, item string) bool {
	return slices.Contains(
		fp.Apply(list, func(e model.ValueListItem) string { return e.Name }),
		item)
}

func ValidateAsset(dto *model.Asset, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !inValueList(valueLists.AssetStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !inValueList(valueLists.AssetTypes, dto.Type)},
		{Name: "Name", Missing: dto.Name == ""},
	})
}

func ValidateCase(dto *model.Case, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Severity", Message: "Invalid value.", Invalid: !inValueList(valueLists.CaseSeverities, dto.Severity)},
		{Name: "Outcome", Message: "Invalid value.", Invalid: !inValueList(valueLists.CaseOutcomes, dto.Outcome)},
		{Name: "SketchID", Message: "Invalid value. Must be positive.", Invalid: dto.SketchID < 0},
	})
}

func ValidateEvent(dto *model.Event, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Time", Missing: dto.Time.IsZero()},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !inValueList(valueLists.EventTypes, dto.Type)},
		{Name: "Event", Missing: dto.Event == ""},
	})
}

func ValidateEvidence(dto *model.Evidence, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Type", Missing: dto.Type == ""},
		{Name: "StartsAt", Message: "Start time is required when end time is set.", Invalid: !dto.EndsAt.IsZero() && dto.StartsAt.IsZero()},
		{Name: "EndsAt", Message: "End time must be after start time.", Invalid: !dto.StartsAt.IsZero() && !dto.EndsAt.IsZero() && time.Time(dto.EndsAt).Before(time.Time(dto.StartsAt))},
	})
}

func ValidateIndicator(dto *model.Indicator, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Value", Missing: dto.Value == ""},
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !inValueList(valueLists.IndicatorStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !inValueList(valueLists.IndicatorTypes, dto.Type)},
		{Name: "TLP", Message: "Invalid value.", Missing: dto.TLP == "", Invalid: !inValueList(valueLists.IndicatorTLPs, dto.TLP)},
	})
}

func ValidateAPIKey(dto *model.APIKey, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !inValueList(valueLists.APIKeyTypes, dto.Type)},
	})
}

func ValidateMalware(dto *model.Malware, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Path", Missing: dto.Path == ""},
		{Name: "Hash", Message: "Invalid hash. Only letters and digits are allowed.", Missing: dto.Hash == "", Invalid: !reAlnum.MatchString(dto.Hash)},
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !inValueList(valueLists.MalwareStatus, dto.Status)},
	})
}

func ValidateNote(dto *model.Note, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Category", Missing: dto.Category == ""},
		{Name: "Title", Missing: dto.Title == ""},
	})
}

func ValidateComment(dto *model.Comment, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Message", Missing: dto.Message == ""},
	})
}

func ValidateTask(dto *model.Task, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "Task", Missing: dto.Task == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !inValueList(valueLists.TaskTypes, dto.Type)},
	})
}

func ValidateReportTemplate(dto *model.ReportTemplate, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{
			Name: "Name", Missing: dto.Name == "",
			// dto.Name is written to files/templates/<Name> (O_TRUNC) and later
			// joined again for download/delete, so a traversing name must be
			// rejected before it ever reaches the disk, not just an extension check.
			Invalid: !isFlatName(dto.Name) ||
				!slices.Contains([]string{".odt", ".ods", ".odp", ".docx"}, filepath.Ext(dto.Name)),
			Message: "Invalid file name or unsupported file type.",
		},
	})
}

func ValidateUser(dto *model.User, valueLists model.ValueLists) valid.ValidationError {
	return valid.Check([]valid.Condition{
		{Name: "ID", Missing: dto.ID == ""},
		{Name: "Role", Message: "Invalid role", Invalid: !inValueList(valueLists.UserRoles, dto.Role)},
	})
}

func ValidateAutomationRule(dto *model.AutomationRule, valueLists model.ValueLists) valid.ValidationError {
	// compile condition
	msg := ""
	_, err := modules.CompileAutomationRule(*dto)
	if err != nil {
		msg = err.Error()
	}

	return valid.Check([]valid.Condition{
		{Name: "ID", Missing: dto.ID == ""},
		{Name: "Trigger", Message: "Invalid trigger", Invalid: !inValueList(valueLists.AutomationRuleTriggers, dto.Trigger)},
		{Name: "Module", Message: "Invalid module", Invalid: !slices.Contains(fp.Keys(modules.Modules), dto.Module)},
		{Name: "Condition", Message: msg, Invalid: err != nil},
	})
}

func ValidateCustomAttribute(dto *model.CustomAttribute, _ model.ValueLists) valid.ValidationError {
	entities := []string{"Case", "Asset", "Event", "Evidence", "Indicator", "Malware", "Note", "Task"}
	types := []string{"string", "textfield", "checkbox", "date", "datetime", "select"}

	return valid.Check([]valid.Condition{
		{Name: "Entity", Invalid: !slices.Contains(entities, dto.Entity), Message: "Invalid entity"},
		{Name: "Label", Missing: dto.Label == ""},
		{Name: "Type", Invalid: !slices.Contains(types, dto.Type), Message: "Invalid type"},
	})
}

func ValidateValueListItem(dto *model.ValueListItem, _ model.ValueLists) valid.ValidationError {
	states := []string{"", "success", "warning", "error"}
	valueLists := []string{"AssetStatus", "AssetTypes", "CaseSeverities", "CaseOutcomes", "EventTypes", "EvidenceTypes", "IndicatorStatus", "IndicatorTypes", "MalwareStatus", "TaskTypes"}

	return valid.Check([]valid.Condition{
		{Name: "ID", Missing: dto.ID == ""},
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Category", Invalid: !slices.Contains(valueLists, dto.Category), Message: "Invalid category"},
		{Name: "State", Invalid: !slices.Contains(states, dto.State), Message: "Invalid state"},
	})
}
