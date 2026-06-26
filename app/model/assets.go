package model

type Asset struct {
	ID     string `gorm:"primaryKey"`
	Status string
	Type   string
	Name   string
	Addr   string
	Notes  string
	CaseID string
	Custom Custom `form:"-"`

	FirstSeen Time `gorm:"->"`
	LastSeen  Time `gorm:"->"`
	Events    int  `gorm:"->"`
}

func (store *Store) ListAssets(cid string) ([]Asset, error) {
	list := []Asset{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("count()")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events", fsq, lsq, evq).
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetAsset(cid string, id string) (Asset, error) {
	obj := Asset{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("count()")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events", fsq, lsq, evq).
		First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) GetAssetByName(cid string, name string) (Asset, error) {
	obj := Asset{}
	fsq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("min(time)")
	lsq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("max(time)")
	evq := store.DB.Table("events").Joins("LEFT JOIN event_assets ON events.id = event_assets.event_id").Where("event_assets.asset_id = assets.id").Select("count()")
	tx := store.DB.
		Select("*, (?) as first_seen, (?) as last_seen, (?) as events", fsq, lsq, evq).
		Where("case_id = ? and name = ?", cid, name).
		First(&obj)
	return obj, tx.Error
}

func (store *Store) SaveAsset(cid string, obj Asset) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteAsset(cid string, id string) error {
	return store.DB.Delete(&Asset{}, "id = ?", id).Error
}
