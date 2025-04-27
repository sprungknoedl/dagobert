package model

import "errors"

type Enums struct {
	AssetStatus     []EnumItem
	AssetTypes      []EnumItem
	CaseSeverities  []EnumItem
	CaseOutcomes    []EnumItem
	EventTypes      []EnumItem
	EvidenceTypes   []EnumItem
	IndicatorStatus []EnumItem
	IndicatorTypes  []EnumItem
	IndicatorTLPs   []EnumItem
	KeyTypes        []EnumItem
	MalwareStatus   []EnumItem
	TaskTypes       []EnumItem
}

type EnumItem struct {
	ID    string
	Group string
	Name  string
	Icon  string
	State string
}

var assetStatus = []EnumItem{
	{Group: "AssetStatus", Name: "Compromised", Icon: "hio-bug-ant", State: "error"},
	{Group: "AssetStatus", Name: "Accessed", Icon: "hio-command-line", State: "warning"},
	{Group: "AssetStatus", Name: "Under investigation", Icon: "", State: ""},
	{Group: "AssetStatus", Name: "No sign of compromise", Icon: "hio-check-circle", State: "success"},
	{Group: "AssetStatus", Name: "Out of scope", Icon: "", State: ""},
}

var assetTypes = []EnumItem{
	{Group: "AssetTypes", Name: "Account", Icon: "hio-user"},
	{Group: "AssetTypes", Name: "Desktop", Icon: "hio-computer-desktop"},
	{Group: "AssetTypes", Name: "Server", Icon: "hio-server"},
	{Group: "AssetTypes", Name: "Other", Icon: "hio-question-mark-circle"},
}

var caseSeverities = []EnumItem{
	{Group: "CaseSeverities", Name: "Low"},
	{Group: "CaseSeverities", Name: "Medium"},
	{Group: "CaseSeverities", Name: "High"},
}

var caseOutcomes = []EnumItem{
	{Group: "CaseOutcomes", Name: ""},
	{Group: "CaseOutcomes", Name: "False positive"},
	{Group: "CaseOutcomes", Name: "True positive"},
	{Group: "CaseOutcomes", Name: "Benign positive"},
}

var eventTypes = []EnumItem{
	{Group: "EventTypes", Name: "Reconnaissance", Icon: "hio-magnifying-glass"},
	{Group: "EventTypes", Name: "Resource Development", Icon: "hio-cog-6-tooth"},
	{Group: "EventTypes", Name: "Initial Access", Icon: "hio-lock-open"},
	{Group: "EventTypes", Name: "Execution", Icon: "hio-play"},
	{Group: "EventTypes", Name: "Persistence", Icon: "hio-arrow-path"},
	{Group: "EventTypes", Name: "Privilege Escalation", Icon: "hio-logout"},
	{Group: "EventTypes", Name: "Defense Evasion", Icon: "hio-eye-slash"},
	{Group: "EventTypes", Name: "Credential Access", Icon: "hio-identification"},
	{Group: "EventTypes", Name: "Discovery", Icon: "hio-eye"},
	{Group: "EventTypes", Name: "Lateral Movement", Icon: "hio-arrows-right-left"},
	{Group: "EventTypes", Name: "Collection", Icon: "hio-arrow-down-tray"},
	{Group: "EventTypes", Name: "C2", Icon: "hio-server"},
	{Group: "EventTypes", Name: "Exfiltration", Icon: "hio-truck"},
	{Group: "EventTypes", Name: "Impact", Icon: "hio-fire"},
	{Group: "EventTypes", Name: "Legitimate", Icon: "hio-check-circle", State: "success"},
	{Group: "EventTypes", Name: "Remediation", Icon: "hio-heart", State: "success"},
	{Group: "EventTypes", Name: "Other"},
}

var evidenceTypes = []EnumItem{
	{Group: "EvidenceTypes", Name: "File", Icon: "hio-document"},
	{Group: "EvidenceTypes", Name: "Logs", Icon: "hio-document-text"},
	{Group: "EvidenceTypes", Name: "Triage", Icon: "hio-archive-box"},
	{Group: "EvidenceTypes", Name: "System Image", Icon: "hio-server"},
	{Group: "EvidenceTypes", Name: "Memory Dump", Icon: "hio-cpu-chip"},
	{Group: "EvidenceTypes", Name: "Malware", Icon: "hio-bug-ant"},
	{Group: "EvidenceTypes", Name: "Other", Icon: "hio-cube"},
}

var indicatorStatus = []EnumItem{
	{Group: "IndicatorStatus", Name: "Confirmed", Icon: "hio-bug-ant", State: "error"},
	{Group: "IndicatorStatus", Name: "Suspicious", Icon: "hio-finger-print", State: "warning"},
	{Group: "IndicatorStatus", Name: "Under investigation", Icon: "", State: ""},
	{Group: "IndicatorStatus", Name: "Unrelated", Icon: "hio-check-circle", State: "success"},
}

var indicatorTypes = []EnumItem{
	{Group: "IndicatorTypes", Name: "IP", Icon: "hio-map-pin"},
	{Group: "IndicatorTypes", Name: "Domain", Icon: "hio-globe-europe-africa"},
	{Group: "IndicatorTypes", Name: "URL", Icon: "hio-link"},
	{Group: "IndicatorTypes", Name: "Path", Icon: "hio-folder-open"},
	{Group: "IndicatorTypes", Name: "Hash", Icon: "hio-finger-print"},
	{Group: "IndicatorTypes", Name: "Service", Icon: "hio-command-line"},
	{Group: "IndicatorTypes", Name: "Other", Icon: "hio-question-mark-circle"},
}

var indicatorTLPs = []EnumItem{
	{Group: "IndicatorTLPs", Name: "TLP:RED", State: "error"},
	{Group: "IndicatorTLPs", Name: "TLP:AMBER", State: "warning"},
	{Group: "IndicatorTLPs", Name: "TLP:GREEN", State: "success"},
	{Group: "IndicatorTLPs", Name: "TLP:CLEAR"},
}

var keyTypes = []EnumItem{
	{Group: "KeyTypes", Name: "API", Icon: "hio-beaker"},
	{Group: "KeyTypes", Name: "Dagobert", Icon: "hio-bolt"},
	{Group: "KeyTypes", Name: "Donald", Icon: "hio-camera"},
}

var malwareStatus = []EnumItem{
	{Group: "MalwareStatus", Name: "Malicious", Icon: "hio-bug-ant", State: "error"},
	{Group: "MalwareStatus", Name: "Suspicious", Icon: "hio-finger-print", State: "warning"},
	{Group: "MalwareStatus", Name: "Under investigation", Icon: "", State: ""},
	{Group: "MalwareStatus", Name: "Unrelated", Icon: "hio-check-circle", State: "success"},
}

var taskTypes = []EnumItem{
	{Group: "TaskTypes", Name: "Information request", Icon: "hio-question-mark-circle"},
	{Group: "TaskTypes", Name: "Analysis", Icon: "hio-magnifying-glass"},
	{Group: "TaskTypes", Name: "Deliverable", Icon: "hio-document-text"},
	{Group: "TaskTypes", Name: "Checkpoint", Icon: "hio-clipboard-document-check"},
	{Group: "TaskTypes", Name: "Other", Icon: "hio-question-mark-circle"},
}

func (store *Store) ListEnums() (Enums, error) {
	return Enums{
		AssetStatus:     assetStatus,
		AssetTypes:      assetTypes,
		CaseSeverities:  caseSeverities,
		CaseOutcomes:    caseOutcomes,
		EventTypes:      eventTypes,
		EvidenceTypes:   evidenceTypes,
		IndicatorStatus: indicatorStatus,
		IndicatorTypes:  indicatorTypes,
		KeyTypes:        keyTypes,
		MalwareStatus:   malwareStatus,
		TaskTypes:       taskTypes,
	}, nil
}

func (store *Store) ListEnumsByGroup(group string) ([]EnumItem, error) {
	return nil, errors.ErrUnsupported
}

func (store *Store) GetEnum(group string, id string) (EnumItem, error) {
	return EnumItem{}, errors.ErrUnsupported
}

func (store *Store) SaveEnum(group string, item EnumItem) (EnumItem, error) {
	return EnumItem{}, errors.ErrUnsupported
}

func (store *Store) DeleteEnum(group string, id string) error {
	return errors.ErrUnsupported
}
