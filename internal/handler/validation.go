package handler

import (
	"regexp"
	"slices"
	"time"

	"github.com/sprungknoedl/dagobert/internal/templ"
	"github.com/sprungknoedl/dagobert/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

var regexIP = regexp.MustCompile(`^$|^(?:\d{1,3}\.){3}\d{1,3}$`)

func ValidateAsset(dto templ.AssetDTO) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.AssetTypes, dto.Type)},
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "IP", Message: "Invalid format, expected e.g. '203.0.113.1'.", Invalid: !regexIP.MatchString(dto.IP)},
		{Name: "Compromised", Message: "Invalid value.", Missing: dto.Compromised == "", Invalid: !slices.Contains(model.AssetCompromised, dto.Compromised)},
	})
}

func ValidateCase(dto templ.CaseDTO) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Missing: dto.Name == ""},
		{Name: "Classification", Missing: dto.Classification == ""},
		{Name: "Severity", Message: "Invalid value.", Missing: dto.Severity == "", Invalid: !slices.Contains(model.CaseSeverities, dto.Severity)},
		{Name: "Outcome", Message: "Invalid value.", Invalid: !slices.Contains(model.CaseOutcomes, dto.Outcome)},
	})
}

func ValidateEvent(dto templ.EventDTO) valid.Result {
	_, terr := time.Parse(time.RFC3339, dto.Time)
	return valid.Check([]valid.Condition{
		{Name: "Time", Message: "Invalid format, expected e.g. '2006-01-02T15:04:05Z'.", Missing: dto.Time == "", Invalid: terr != nil},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.EventTypes, dto.Type)},
		{Name: "Event System", Message: "Invalid asset.", Missing: dto.AssetA == ""},
		{Name: "Remote System", Message: "Invalid asset."},
		{Name: "Direction", Message: "Invalid type.", Invalid: !slices.Contains(model.EventDirections, dto.Direction)},
		{Name: "Event", Missing: dto.Event == ""},
	})
}

func ValidateEvidence(dto templ.EvidenceDTO) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Name", Message: "Invalid name.", Missing: dto.Name == "", Invalid: dto.Name == "."},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.EvidenceTypes, dto.Type)},
	})
}

func ValidateIndicator(dto templ.IndicatorDTO) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Value", Missing: dto.Value == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.IndicatorTypes, dto.Type)},
		{Name: "TLP", Message: "Invalid value.", Missing: dto.Type == "", Invalid: !slices.Contains(model.IndicatorTLPs, dto.TLP)},
	})
}

func ValidateMalware(dto templ.MalwareDTO) valid.Result {
	_, cerr := time.Parse(time.RFC3339, dto.CDate)
	_, merr := time.Parse(time.RFC3339, dto.MDate)
	return valid.Check([]valid.Condition{
		{Name: "Filename", Missing: dto.Filename == ""},
		{Name: "System", Missing: dto.System == ""},
		{Name: "CDate", Message: "Invalid format, expected e.g. '2006-01-02T15:04:05Z'.", Invalid: dto.CDate != "" && cerr != nil},
		{Name: "MDate", Message: "Invalid format, expected e.g. '2006-01-02T15:04:05Z'.", Invalid: dto.MDate != "" && merr != nil},
	})
}

func ValidateNote(dto templ.NoteDTO) valid.Result {
	return valid.Check([]valid.Condition{
		{Name: "Title", Missing: dto.Title == ""},
		{Name: "Description", Missing: dto.Description == ""},
	})
}

func ValidateTask(dto templ.TaskDTO) valid.Result {
	_, terr := time.Parse("2006-01-02", dto.DateDue)
	return valid.Check([]valid.Condition{
		{Name: "Task", Missing: dto.Task == ""},
		{Name: "Type", Message: "Invalid type.", Missing: dto.Type == "", Invalid: !slices.Contains(model.TaskTypes, dto.Type)},
		{Name: "DateDue", Message: "Invalid format, expected e.g. '2006-01-02'.", Invalid: dto.DateDue != "" && terr != nil},
	})
}
