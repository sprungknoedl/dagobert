package model

import (
	"time"
)

var CaseSeverities = []string{"Low", "Medium", "High"}
var CaseOutcomes = []string{"False positive", "True positive", "Benign positive"}

type Case struct {
	ID             int64  `json:"id" gorm:"primarykey"`
	Name           string `json:"name" binding:"required"`
	Closed         bool   `json:"closed"`
	Classification string `json:"classification"`
	Severity       string `json:"severity" binding:"required"`
	Outcome        string `json:"outcome"`
	Summary        string `json:"summary"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`

	Assets     []Asset     `json:"-" binding:"-"`
	Evidences  []Evidence  `json:"-" binding:"-"`
	Indicators []Indicator `json:"-" binding:"-"`
	Events     []Event     `json:"-" binding:"-"`
	Malware    []Malware   `json:"-" binding:"-"`
	Notes      []Note      `json:"-" binding:"-"`
	Tasks      []Task      `json:"-" binding:"-"`
	Users      []User      `json:"-" binding:"-"`
}

var EvidenceTypes = []string{"File", "Log", "Artifacts Collection", "System Image", "Memory Dump", "Other"}

type Evidence struct {
	ID          int64  `json:"id" gorm:"primarykey"`
	Type        string `json:"type" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	Hash        string `json:"hash"`
	Location    string `json:"location"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var AssetTypes = []string{"Account", "Desktop", "Server", "Other"}
var AssetCompromised = []string{"Compromised", "Not compromised", "Unknown"}

type Asset struct {
	ID          int64  `json:"id" gorm:"primarykey"`
	Type        string `json:"type" binding:"required"`
	Name        string `json:"name" binding:"required"`
	IP          string `json:"ip"`
	Description string `json:"description"`
	Compromised string `json:"compromised"`
	Analysed    bool   `json:"analysed"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var IndicatorTypes = []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"}
var IndicatorTLPs = []string{"TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"}

type Indicator struct {
	ID          int64  `json:"id" gorm:"primarykey"`
	Type        string `json:"type" binding:"required"`
	Value       string `json:"value" binding:"required"`
	TLP         string `json:"tlp" binding:"required"`
	Description string `json:"description"`
	Source      string `json:"source"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var EventTypes = []string{"Event Log", "File", "Human", "Lateral Movement", "Exfiltration", "Malware", "C2", "DFIR", "Other"}
var EventDirections = []string{"", "→", "←"}

type Event struct {
	ID        int64     `json:"id" gorm:"primarykey"`
	Time      time.Time `json:"time" binding:"required"`
	Type      string    `json:"type" binding:"required"`
	AssetA    string    `json:"assetA" binding:"required"`
	AssetB    string    `json:"assetB"`
	Direction string    `json:"direction"`
	Event     string    `json:"event" binding:"required"`
	Raw       string    `json:"raw"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

type Malware struct {
	ID       int64     `json:"id" gorm:"primarykey"`
	Filename string    `json:"filename" binding:"required"`
	Filepath string    `json:"filepath"`
	CDate    time.Time `json:"cDate"`
	MDate    time.Time `json:"mDate"`
	System   string    `json:"system"`
	Hash     string    `json:"hash"`
	Notes    string    `json:"notes"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

type Note struct {
	ID          int64  `json:"id" gorm:"primarykey"`
	Title       string `json:"title" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description" binding:"required"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var TaskTypes = []string{"Information request", "Analysis", "Deliverable", "Checkpoint", "Other"}

type Task struct {
	ID      int64     `json:"id" gorm:"primarykey"`
	Type    string    `json:"type" binding:"required"`
	Task    string    `json:"task" binding:"required"`
	Done    bool      `json:"done"`
	Owner   string    `json:"owner"`
	DateDue time.Time `json:"dateDue,omitempty"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded,omitempty"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

type User struct {
	ID      int64  `json:"id" gorm:"primarykey"`
	Name    string `json:"name"`
	Company string `json:"company"`
	Role    string `json:"role"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Notes   string `json:"notes"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-" binding:"-"`

	DateAdded    time.Time `json:"dateAdded"`
	DateModified time.Time `json:"dateModified"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}
