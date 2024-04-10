package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.AssetStore = &Store{}

func (store *Store) ListAssets(cid ulid.ULID) ([]model.Asset, error) {
	var list []model.Asset
	result := store.db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindAssets(cid ulid.ULID, search string, sort string) ([]model.Asset, error) {
	var list []model.Asset
	query := store.db.
		Where("case_id = ?", cid).
		Where(store.db.
			Where("instr(type, ?) > 0", search).
			Or("instr(name, ?) > 0", search).
			Or("instr(ip, ?) > 0", search).
			Or("instr(description, ?) > 0", search).
			Or("instr(compromised, ?) > 0", search))

	switch sort {
	case "analysed":
		query = query.Order("analysed asc, name asc")
	case "-analysed":
		query = query.Order("analysed desc, name asc")
	case "compromised":
		query = query.Order("compromised asc, name asc")
	case "-compromised":
		query = query.Order("compromised desc, name asc")
	case "desc":
		query = query.Order("description asc, name asc")
	case "-desc":
		query = query.Order("description desc, name asc")
	case "ip":
		query = query.Order("ip asc, name asc")
	case "-ip":
		query = query.Order("ip desc, name asc")
	case "type":
		query = query.Order("type asc, name asc")
	case "-type":
		query = query.Order("type desc, name asc")
	case "-name":
		query = query.Order("name desc")
	default: // case "name"
		query = query.Order("name asc")
	}

	result := query.Find(&list)
	return list, result.Error
}

func (store *Store) GetAsset(cid ulid.ULID, id ulid.ULID) (model.Asset, error) {
	x := model.Asset{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) GetAssetByName(cid ulid.ULID, name string) (model.Asset, error) {
	x := model.Asset{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "name = ?", name)
	return x, result.Error
}

func (store *Store) SaveAsset(cid ulid.ULID, x model.Asset) (model.Asset, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteAsset(cid ulid.ULID, id ulid.ULID) error {
	x := model.Asset{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
