package model

import "github.com/oklog/ulid/v2"

type CaseStore interface {
	FindCases(search string, sort string) ([]Case, error)
	ListCases() ([]Case, error)
	GetCase(id ulid.ULID) (Case, error)
	GetCaseFull(id ulid.ULID) (Case, error)
	SaveCase(x Case) (Case, error)
	DeleteCase(id ulid.ULID) error
}

type EventStore interface {
	FindEvents(cid ulid.ULID, search string, sort string) ([]Event, error)
	ListEvents(cid ulid.ULID) ([]Event, error)
	GetEvent(cid ulid.ULID, id ulid.ULID) (Event, error)
	SaveEvent(cid ulid.ULID, x Event) (Event, error)
	DeleteEvent(cid ulid.ULID, id ulid.ULID) error
}

type AssetStore interface {
	ListAssets(cid ulid.ULID) ([]Asset, error)
	FindAssets(cid ulid.ULID, search string, sort string) ([]Asset, error)
	GetAsset(cid ulid.ULID, id ulid.ULID) (Asset, error)
	GetAssetByName(cid ulid.ULID, name string) (Asset, error)
	SaveAsset(cid ulid.ULID, x Asset) (Asset, error)
	DeleteAsset(cid ulid.ULID, id ulid.ULID) error
}

type MalwareStore interface {
	ListMalware(cid ulid.ULID) ([]Malware, error)
	FindMalware(cid ulid.ULID, search string, sort string) ([]Malware, error)
	GetMalware(cid ulid.ULID, id ulid.ULID) (Malware, error)
	SaveMalware(cid ulid.ULID, x Malware) (Malware, error)
	DeleteMalware(cid ulid.ULID, id ulid.ULID) error
}

type IndicatorStore interface {
	ListIndicators(cid ulid.ULID) ([]Indicator, error)
	FindIndicators(cid ulid.ULID, search string, sort string) ([]Indicator, error)
	GetIndicator(cid ulid.ULID, id ulid.ULID) (Indicator, error)
	SaveIndicator(cid ulid.ULID, x Indicator) (Indicator, error)
	DeleteIndicator(cid ulid.ULID, id ulid.ULID) error
}

type UserStore interface {
	ListUsers() ([]User, error)
	FindUsers(search string, sort string) ([]User, error)
	GetUser(id ulid.ULID) (User, error)
	SaveUser(x User) (User, error)
	DeleteUser(id ulid.ULID) error
}

type EvidenceStore interface {
	ListEvidences(cid ulid.ULID) ([]Evidence, error)
	FindEvidences(cid ulid.ULID, search string, sort string) ([]Evidence, error)
	GetEvidence(cid ulid.ULID, id ulid.ULID) (Evidence, error)
	SaveEvidence(cid ulid.ULID, x Evidence) (Evidence, error)
	DeleteEvidence(cid ulid.ULID, id ulid.ULID) error
}

type TaskStore interface {
	ListTasks(cid ulid.ULID) ([]Task, error)
	FindTasks(cid ulid.ULID, search string, sort string) ([]Task, error)
	GetTask(cid ulid.ULID, id ulid.ULID) (Task, error)
	SaveTask(cid ulid.ULID, x Task) (Task, error)
	DeleteTask(cid ulid.ULID, id ulid.ULID) error
}

type NoteStore interface {
	ListNotes(cid ulid.ULID) ([]Note, error)
	FindNotes(cid ulid.ULID, search string, sort string) ([]Note, error)
	GetNote(cid ulid.ULID, id ulid.ULID) (Note, error)
	SaveNote(cid ulid.ULID, x Note) (Note, error)
	DeleteNote(cid ulid.ULID, id ulid.ULID) error
}
