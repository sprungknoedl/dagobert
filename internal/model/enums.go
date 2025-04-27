package model

type EnumItem struct {
	Group string `json:"group"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	State string `json:"state"`
}

var AssetStatus = []EnumItem{
	{Group: "AssetStatus", Name: "Compromised", Icon: "hio-bug-ant", State: "error"},
	{Group: "AssetStatus", Name: "Accessed", Icon: "hio-command-line", State: "warning"},
	{Group: "AssetStatus", Name: "Under investigation", Icon: "", State: ""},
	{Group: "AssetStatus", Name: "No sign of compromise", Icon: "hio-check-circle", State: "success"},
	{Group: "AssetStatus", Name: "Out of scope", Icon: "", State: ""},
}

var AssetTypes = []EnumItem{
	{Group: "AssetTypes", Name: "Account", Icon: "hio-user"},
	{Group: "AssetTypes", Name: "Desktop", Icon: "hio-computer-desktop"},
	{Group: "AssetTypes", Name: "Server", Icon: "hio-server"},
	{Group: "AssetTypes", Name: "Other", Icon: "hio-question-mark-circle"},
}

var CaseSeverities = []EnumItem{
	{Group: "CaseSeverities", Name: "Low"},
	{Group: "CaseSeverities", Name: "Medium"},
	{Group: "CaseSeverities", Name: "High"},
}

var CaseOutcomes = []EnumItem{
	{Group: "CaseOutcomes", Name: ""},
	{Group: "CaseOutcomes", Name: "False positive"},
	{Group: "CaseOutcomes", Name: "True positive"},
	{Group: "CaseOutcomes", Name: "Benign positive"},
}

var EventTypes = []EnumItem{
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

var EvidenceTypes = []EnumItem{
	{Group: "EvidenceTypes", Name: "File", Icon: "hio-document"},
	{Group: "EvidenceTypes", Name: "Logs", Icon: "hio-document-text"},
	{Group: "EvidenceTypes", Name: "Triage", Icon: "hio-archive-box"},
	{Group: "EvidenceTypes", Name: "System Image", Icon: "hio-server"},
	{Group: "EvidenceTypes", Name: "Memory Dump", Icon: "hio-cpu-chip"},
	{Group: "EvidenceTypes", Name: "Malware", Icon: "hio-bug-ant"},
	{Group: "EvidenceTypes", Name: "Other", Icon: "hio-cube"},
}

var IndicatorStatus = []EnumItem{
	{Group: "IndicatorStatus", Name: "Confirmed", Icon: "hio-bug-ant", State: "error"},
	{Group: "IndicatorStatus", Name: "Suspicious", Icon: "hio-finger-print", State: "warning"},
	{Group: "IndicatorStatus", Name: "Under investigation", Icon: "", State: ""},
	{Group: "IndicatorStatus", Name: "Unrelated", Icon: "hio-check-circle", State: "success"},
}

var IndicatorTypes = []EnumItem{
	{Group: "IndicatorTypes", Name: "IP", Icon: "hio-map-pin"},
	{Group: "IndicatorTypes", Name: "Domain", Icon: "hio-globe-europe-africa"},
	{Group: "IndicatorTypes", Name: "URL", Icon: "hio-link"},
	{Group: "IndicatorTypes", Name: "Path", Icon: "hio-folder-open"},
	{Group: "IndicatorTypes", Name: "Hash", Icon: "hio-finger-print"},
	{Group: "IndicatorTypes", Name: "Service", Icon: "hio-command-line"},
	{Group: "IndicatorTypes", Name: "Other", Icon: "hio-question-mark-circle"},
}

var IndicatorTLPs = []EnumItem{
	{Group: "IndicatorTLPs", Name: "TLP:RED", State: "error"},
	{Group: "IndicatorTLPs", Name: "TLP:AMBER", State: "warning"},
	{Group: "IndicatorTLPs", Name: "TLP:GREEN", State: "success"},
	{Group: "IndicatorTLPs", Name: "TLP:CLEAR"},
}

var KeyTypes = []EnumItem{
	{Group: "KeyTypes", Name: "API", Icon: "hio-beaker"},
	{Group: "KeyTypes", Name: "Dagobert", Icon: "hio-bolt"},
	{Group: "KeyTypes", Name: "Donald", Icon: "hio-camera"},
}

var MalwareStatus = []EnumItem{
	{Group: "MalwareStatus", Name: "Malicious", Icon: "hio-bug-ant", State: "error"},
	{Group: "MalwareStatus", Name: "Suspicious", Icon: "hio-finger-print", State: "warning"},
	{Group: "MalwareStatus", Name: "Under investigation", Icon: "", State: ""},
	{Group: "MalwareStatus", Name: "Unrelated", Icon: "hio-check-circle", State: "success"},
}

var TaskTypes = []EnumItem{
	{Group: "TaskTypes", Name: "Information request", Icon: "hio-question-mark-circle"},
	{Group: "TaskTypes", Name: "Analysis", Icon: "hio-magnifying-glass"},
	{Group: "TaskTypes", Name: "Deliverable", Icon: "hio-document-text"},
	{Group: "TaskTypes", Name: "Checkpoint", Icon: "hio-clipboard-document-check"},
	{Group: "TaskTypes", Name: "Other", Icon: "hio-question-mark-circle"},
}
