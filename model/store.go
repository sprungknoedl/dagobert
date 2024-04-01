package model

type CaseStore interface {
	FindCases(search string, sort string) ([]Case, error)
	ListCases() ([]Case, error)
	GetCase(id int64) (Case, error)
	GetCaseFull(id int64) (Case, error)
	SaveCase(x Case) (Case, error)
	DeleteCase(id int64) error
}

type EventStore interface {
	FindEvents(cid int64, search string, sort string) ([]Event, error)
	ListEvents(cid int64) ([]Event, error)
	GetEvent(cid int64, id int64) (Event, error)
	SaveEvent(cid int64, x Event) (Event, error)
	DeleteEvent(cid int64, id int64) error
}

type AssetStore interface {
	ListAssets(cid int64) ([]Asset, error)
	FindAssets(cid int64, search string, sort string) ([]Asset, error)
	GetAsset(cid int64, id int64) (Asset, error)
	GetAssetByName(cid int64, name string) (Asset, error)
	SaveAsset(cid int64, x Asset) (Asset, error)
	DeleteAsset(cid int64, id int64) error
}

type MalwareStore interface {
	ListMalware(cid int64) ([]Malware, error)
	FindMalware(cid int64, search string, sort string) ([]Malware, error)
	GetMalware(cid int64, id int64) (Malware, error)
	SaveMalware(cid int64, x Malware) (Malware, error)
	DeleteMalware(cid int64, id int64) error
}

type IndicatorStore interface {
	ListIndicators(cid int64) ([]Indicator, error)
	FindIndicators(cid int64, search string, sort string) ([]Indicator, error)
	GetIndicator(cid int64, id int64) (Indicator, error)
	SaveIndicator(cid int64, x Indicator) (Indicator, error)
	DeleteIndicator(cid int64, id int64) error
}

type UserStore interface {
	ListUsers() ([]User, error)
	FindUsers(search string, sort string) ([]User, error)
	GetUser(id int64) (User, error)
	SaveUser(x User) (User, error)
	DeleteUser(id int64) error
}

type EvidenceStore interface {
	ListEvidences(cid int64) ([]Evidence, error)
	FindEvidences(cid int64, search string, sort string) ([]Evidence, error)
	GetEvidence(cid int64, id int64) (Evidence, error)
	SaveEvidence(cid int64, x Evidence) (Evidence, error)
	DeleteEvidence(cid int64, id int64) error
}

type TaskStore interface {
	ListTasks(cid int64) ([]Task, error)
	FindTasks(cid int64, search string, sort string) ([]Task, error)
	GetTask(cid int64, id int64) (Task, error)
	SaveTask(cid int64, x Task) (Task, error)
	DeleteTask(cid int64, id int64) error
}

type NoteStore interface {
	ListNotes(cid int64) ([]Note, error)
	FindNotes(cid int64, search string, sort string) ([]Note, error)
	GetNote(cid int64, id int64) (Note, error)
	SaveNote(cid int64, x Note) (Note, error)
	DeleteNote(cid int64, id int64) error
}
