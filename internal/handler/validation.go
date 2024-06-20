package handler

import (
	"regexp"
	"slices"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

var regexIP = regexp.MustCompile(`^$|^(?:\d{1,3}\.){3}\d{1,3}$`)

func ValidateAsset(dto model.Asset) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Status", Message: "Invalid status.", Missing: dto.Status == "", Invalid: !slices.Contains(model.AssetStatus, dto.Status)},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.AssetTypes, dto.Type)},
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "IP", Message: "Invalid format, expected e.g. '203.0.113.1'.", Invalid: !regexIP.MatchString(dto.Addr)},
	})
}

func ValidateCase(dto model.Case) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Severity", Message: "Invalid value.", Invalid: !slices.Contains(model.CaseSeverities, dto.Severity)},
		{Name: "Outcome", Message: "Invalid value.", Invalid: !slices.Contains(model.CaseOutcomes, dto.Outcome)},
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
		{Name: "Name", Message: "Invalid name.", Missing: dto.Name == "", Invalid: dto.Name == "."},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.EvidenceTypes, dto.Type)},
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
	})
}

func ValidateMalware(dto model.Malware) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
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
