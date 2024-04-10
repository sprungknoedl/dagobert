package model

import (
	"time"

	"github.com/oklog/ulid/v2"
)

var CaseSeverities = []string{"Low", "Medium", "High"}
var CaseOutcomes = []string{"", "False positive", "True positive", "Benign positive"}

type Case struct {
	ID             ulid.ULID `gorm:"primaryKey"`
	Name           string
	Closed         bool
	Classification string
	Severity       string
	Outcome        string
	Summary        string

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string

	Assets     []Asset
	Evidences  []Evidence
	Indicators []Indicator
	Events     []Event
	Malware    []Malware
	Notes      []Note
	Tasks      []Task
}

var EvidenceTypes = []string{"File", "Logs", "Artifacts Collection", "System Image", "Memory Dump", "Other"}

type Evidence struct {
	ID          ulid.ULID `gorm:"primaryKey"`
	Type        string
	Name        string
	Description string
	Size        int64
	Hash        string
	Location    string

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

var AssetTypes = []string{"Account", "Desktop", "Server", "Other"}
var AssetCompromised = []string{"Compromised", "Not compromised", "Unknown"}

type Asset struct {
	ID          ulid.ULID `gorm:"primaryKey"`
	Type        string
	Name        string
	IP          string
	Description string
	Compromised string
	Analysed    bool

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

var IndicatorTypes = []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"}
var IndicatorTLPs = []string{"TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"}

type Indicator struct {
	ID          ulid.ULID `gorm:"primaryKey"`
	Type        string
	Value       string
	TLP         string
	Description string
	Source      string

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

var EventTypes = []string{
	"Reconnaissance",
	"Resource Development",
	"Initial Access",
	"Execution",
	"Persistence",
	"Privilege Escalation",
	"Defense Evasion",
	"Credential Access",
	"Discovery",
	"Lateral Movement",
	"Collection",
	"C2",
	"Exfiltration",
	"Impact",
	"DFIR",
	"Other",
}
var EventDirections = []string{"", "→", "←"}

type Event struct {
	ID        ulid.ULID `gorm:"primaryKey"`
	Time      time.Time
	Type      string
	AssetA    string
	AssetB    string
	Direction string
	Event     string
	Raw       string
	KeyEvent  bool

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

type Malware struct {
	ID       ulid.ULID `gorm:"primaryKey"`
	Filename string
	Filepath string
	CDate    time.Time
	MDate    time.Time
	System   string
	Hash     string
	Notes    string

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

type Note struct {
	ID          ulid.ULID `gorm:"primaryKey"`
	Title       string
	Category    string
	Description string

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

var TaskTypes = []string{"Information request", "Analysis", "Deliverable", "Checkpoint", "Other"}

type Task struct {
	ID      ulid.ULID `gorm:"primaryKey"`
	Type    string
	Task    string
	Done    bool
	Owner   string
	DateDue time.Time

	CaseID ulid.ULID
	Case   Case

	DateAdded    time.Time
	DateModified time.Time
	UserAdded    string
	UserModified string
}

type User struct {
	ID    string `gorm:"primaryKey"`
	Name  string
	UPN   string
	Email string
}
