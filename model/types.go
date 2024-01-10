package model

import (
	"time"
)

var CaseSeverities = []string{"Low", "Medium", "High"}
var CaseOutcomes = []string{"", "False positive", "True positive", "Benign positive"}

type Case struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Name           string `json:"name"`
	Closed         bool   `json:"closed"`
	Classification string `json:"classification"`
	Severity       string `json:"severity"`
	Outcome        string `json:"outcome"`
	Summary        string `json:"summary"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`

	Assets     []Asset     `json:"-"`
	Evidences  []Evidence  `json:"-"`
	Indicators []Indicator `json:"-"`
	Events     []Event     `json:"-"`
	Malware    []Malware   `json:"-"`
	Notes      []Note      `json:"-"`
	Tasks      []Task      `json:"-"`
	Users      []User      `json:"-"`
}

var EvidenceTypes = []string{"File", "Log", "Artifacts Collection", "System Image", "Memory Dump", "Other"}

type Evidence struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	Hash        string `json:"hash"`
	Location    string `json:"location"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var AssetTypes = []string{"Account", "Desktop", "Server", "Other"}
var AssetCompromised = []string{"Compromised", "Not compromised", "Unknown"}

type Asset struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Type        string `json:"type"`
	Name        string `json:"name"`
	IP          string `json:"ip"`
	Description string `json:"description"`
	Compromised string `json:"compromised"`
	Analysed    bool   `json:"analysed"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var IndicatorTypes = []string{"IP", "Domain", "URL", "Path", "Hash", "Service", "Other"}
var IndicatorTLPs = []string{"TLP:RED", "TLP:AMBER", "TLP:GREEN", "TLP:CLEAR"}

type Indicator struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Type        string `json:"type"`
	Value       string `json:"value"`
	TLP         string `json:"tlp"`
	Description string `json:"description"`
	Source      string `json:"source"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var EventTypes = []string{"Event Log", "File", "Human", "Lateral Movement", "Exfiltration", "Malware", "C2", "DFIR", "Other"}
var EventDirections = []string{"", "→", "←"}

type Event struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	AssetA    string    `json:"assetA"`
	AssetB    string    `json:"assetB"`
	Direction string    `json:"direction"`
	Event     string    `json:"event"`
	Raw       string    `json:"raw"`
	KeyEvent  bool      `json:"keyevent"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

type Malware struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Filename string    `json:"filename"`
	Filepath string    `json:"filepath"`
	CDate    time.Time `json:"cDate"`
	MDate    time.Time `json:"mDate"`
	System   string    `json:"system"`
	Hash     string    `json:"hash"`
	Notes    string    `json:"notes"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

type Note struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Title       string `json:"title"`
	Category    string `json:"category"`
	Description string `json:"description"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

var TaskTypes = []string{"Information request", "Analysis", "Deliverable", "Checkpoint", "Other"}

type Task struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Type    string    `json:"type"`
	Task    string    `json:"task"`
	Done    bool      `json:"done"`
	Owner   string    `json:"owner"`
	DateDue time.Time `json:"dateDue,omitempty"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded,omitempty"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}

type User struct {
	ID  int64  `json:"id" gorm:"primaryKey"`
	CRC string `json:"-" gorm:"unique"`

	Name    string `json:"name"`
	Company string `json:"company"`
	Role    string `json:"role"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Notes   string `json:"notes"`

	CaseID int64 `json:"caseId"`
	Case   Case  `json:"-"`

	DateAdded    time.Time `json:"dateAdded" gorm:"<-:create"`
	DateModified time.Time `json:"dateModified" gorm:"<-:create"`
	UserAdded    string    `json:"userAdded"`
	UserModified string    `json:"userModified"`
}
