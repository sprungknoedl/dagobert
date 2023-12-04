package main

import (
	"time"
)

type Case struct {
	ID             uint   `json:"id" gorm:"primarykey"`
	Name           string `json:"name" binding:"required"`
	Classification string `json:"classification"`
	Summary        string `json:"summary"`
}

type Evidence struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	Type        string `json:"type" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Size        int    `json:"size"`
	Hash        string `json:"hash"`
	Location    string `json:"location"`
}

type Asset struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	Type        string `json:"type" binding:"required"`
	Name        string `json:"name" binding:"required"`
	IP          string `json:"ip"`
	Description string `json:"description"`
	Compromised string `json:"compromised"`
	Analysed    bool   `json:"analysed"`
}

type Indicator struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	Type        string `json:"type" binding:"required"`
	Value       string `json:"value" binding:"required"`
	TLP         string `json:"tlp" binding:"required"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type Event struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Time      time.Time `json:"time" binding:"required"`
	Type      string    `json:"type" binding:"required"`
	AssetA    string    `json:"assetA" binding:"required"`
	AssetB    string    `json:"assetB"`
	Direction string    `json:"direction"`
	Event     string    `json:"event" binding:"required"`
	Raw       string    `json:"raw"`
}

type Malware struct {
	ID       uint      `json:"id" gorm:"primarykey"`
	Filename string    `json:"filename" binding:"required"`
	Filepath string    `json:"filepath"`
	CDate    time.Time `json:"cDate"`
	MDate    time.Time `json:"mDate"`
	System   string    `json:"system"`
	Hash     string    `json:"hash"`
	Notes    string    `json:"notes"`
}

type Note struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	Title       string `json:"title" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type Task struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Type      string    `json:"type" binding:"required"`
	Task      string    `json:"task" binding:"required"`
	Done      bool      `json:"done"`
	Owner     string    `json:"owner"`
	DateAdded time.Time `json:"dateAdded,omitempty"`
	DateDue   time.Time `json:"dateDue,omitempty"`
}

type User struct {
	ID        uint   `json:"id" gorm:"primarykey"`
	ShortName string `json:"shortName"`
	FullName  string `json:"fullName"`
	Company   string `json:"company"`
	Role      string `json:"role"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Notes     string `json:"notes"`
}
