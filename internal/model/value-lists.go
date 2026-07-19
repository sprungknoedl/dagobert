package model

type ValueLists struct {
	AssetStatus     []ValueListItem
	AssetTypes      []ValueListItem
	CaseSeverities  []ValueListItem
	CaseOutcomes    []ValueListItem
	EventTypes      []ValueListItem
	EvidenceTypes   []ValueListItem
	IndicatorStatus []ValueListItem
	IndicatorTypes  []ValueListItem
	IndicatorTLPs   []ValueListItem
	APIKeyTypes     []ValueListItem
	MalwareStatus   []ValueListItem
	TaskTypes       []ValueListItem

	UserRoles              []ValueListItem
	AutomationRuleTriggers []ValueListItem
}

type ValueListItem struct {
	ID       string `gorm:"primaryKey"`
	Category string `gorm:"<-:create"`
	Rank     int
	Name     string
	Icon     string
	State    string

	// CreatedAt time.Time
	// UpdatedAt time.Time
}

var automationRuleTriggers = []ValueListItem{
	{Name: "OnEvidenceAdded"},
	{Name: "OnIndicatorAdded"},
	{Name: "OnCaseAdded"},
	{Name: "OnCaseUpdated"},
}

var userRoles = []ValueListItem{
	{Name: "Administrator"},
	{Name: "User"},
	{Name: "Read-Only"},
}

func (store *Store) ListValueLists() (ValueLists, error) {
	store.valueListsMu.Lock()
	defer store.valueListsMu.Unlock()
	if store.valueListsCache != nil {
		return *store.valueListsCache, nil
	}

	// fetch valueLists
	list := []ValueListItem{}
	tx := store.DB.Order("category, rank, name asc").Find(&list)
	if tx.Error != nil {
		return ValueLists{}, tx.Error
	}

	// group valueLists
	valueLists := ValueLists{AutomationRuleTriggers: automationRuleTriggers, UserRoles: userRoles, APIKeyTypes: APIKeyTypeValues()}
	for _, item := range list {
		switch item.Category {
		case "AssetStatus":
			valueLists.AssetStatus = append(valueLists.AssetStatus, item)
		case "AssetTypes":
			valueLists.AssetTypes = append(valueLists.AssetTypes, item)
		case "CaseSeverities":
			valueLists.CaseSeverities = append(valueLists.CaseSeverities, item)
		case "CaseOutcomes":
			valueLists.CaseOutcomes = append(valueLists.CaseOutcomes, item)
		case "EventTypes":
			valueLists.EventTypes = append(valueLists.EventTypes, item)
		case "EvidenceTypes":
			valueLists.EvidenceTypes = append(valueLists.EvidenceTypes, item)
		case "IndicatorStatus":
			valueLists.IndicatorStatus = append(valueLists.IndicatorStatus, item)
		case "IndicatorTypes":
			valueLists.IndicatorTypes = append(valueLists.IndicatorTypes, item)
		case "IndicatorTLPs":
			valueLists.IndicatorTLPs = append(valueLists.IndicatorTLPs, item)
		case "MalwareStatus":
			valueLists.MalwareStatus = append(valueLists.MalwareStatus, item)
		case "TaskTypes":
			valueLists.TaskTypes = append(valueLists.TaskTypes, item)
		}
	}

	store.valueListsCache = &valueLists
	return valueLists, nil
}

func (store *Store) GetEnum(id string) (ValueListItem, error) {
	obj := ValueListItem{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) SaveEnum(obj ValueListItem) error {
	err := store.DB.Save(obj).Error
	store.valueListsMu.Lock()
	store.valueListsCache = nil
	store.valueListsMu.Unlock()
	return err
}

func (store *Store) DeleteEnum(id string) error {
	err := store.DB.Delete(&ValueListItem{}, "id = ?", id).Error
	store.valueListsMu.Lock()
	store.valueListsCache = nil
	store.valueListsMu.Unlock()
	return err
}
