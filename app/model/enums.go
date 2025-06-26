package model

type Enums struct {
	AssetStatus     []Enum
	AssetTypes      []Enum
	CaseSeverities  []Enum
	CaseOutcomes    []Enum
	EventTypes      []Enum
	EvidenceTypes   []Enum
	IndicatorStatus []Enum
	IndicatorTypes  []Enum
	IndicatorTLPs   []Enum
	KeyTypes        []Enum
	MalwareStatus   []Enum
	TaskTypes       []Enum

	UserRoles   []Enum
	HookTrigger []Enum
}

type Enum struct {
	ID       string `gorm:"primaryKey"`
	Category string `gorm:"<-:create"`
	Rank     int
	Name     string
	Icon     string
	State    string

	// CreatedAt time.Time
	// UpdatedAt time.Time
}

var hookTrigger = []Enum{
	{Name: "OnEvidenceAdded"},
}

var userRoles = []Enum{
	{Name: "Administrator"},
	{Name: "User"},
	{Name: "Read-Only"},
}

func (store *Store) ListEnums() (Enums, error) {
	// fetch enums
	list := []Enum{}
	tx := store.DB.Order("category, rank, name asc").Find(&list)
	if tx.Error != nil {
		return Enums{}, tx.Error
	}

	// group enums
	enums := Enums{HookTrigger: hookTrigger, UserRoles: userRoles}
	for _, enum := range list {
		switch enum.Category {
		case "AssetStatus":
			enums.AssetStatus = append(enums.AssetStatus, enum)
		case "AssetTypes":
			enums.AssetTypes = append(enums.AssetTypes, enum)
		case "CaseSeverities":
			enums.CaseSeverities = append(enums.CaseSeverities, enum)
		case "CaseOutcomes":
			enums.CaseOutcomes = append(enums.CaseOutcomes, enum)
		case "EventTypes":
			enums.EventTypes = append(enums.EventTypes, enum)
		case "EvidenceTypes":
			enums.EvidenceTypes = append(enums.EvidenceTypes, enum)
		case "IndicatorStatus":
			enums.IndicatorStatus = append(enums.IndicatorStatus, enum)
		case "IndicatorTypes":
			enums.IndicatorTypes = append(enums.IndicatorTypes, enum)
		case "IndicatorTLPs":
			enums.IndicatorTLPs = append(enums.IndicatorTLPs, enum)
		case "MalwareStatus":
			enums.MalwareStatus = append(enums.MalwareStatus, enum)
		case "TaskTypes":
			enums.TaskTypes = append(enums.TaskTypes, enum)
		}
	}

	return enums, nil
}

func (store *Store) GetEnum(id string) (Enum, error) {
	obj := Enum{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveEnum(obj Enum) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteEnum(id string) error {
	return store.DB.Delete(&Enum{}, "id = ?", id).Error
}
