package sqlite

import (
	"github.com/sprungknoedl/dagobert/pkg/model"
	"gorm.io/gorm/clause"
)

var _ model.AssetStore = &Store{}

func (store *Store) ListAssets(cid int64) ([]model.Asset, error) {
	var list []model.Asset
	result := store.db.
		Where("case_id = ?", cid).
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindAssets(cid int64, search string, sort string) ([]model.Asset, error) {
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

func (store *Store) GetAsset(cid int64, id int64) (model.Asset, error) {
	x := model.Asset{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, id)
	return x, result.Error
}

func (store *Store) GetAssetByName(cid int64, name string) (model.Asset, error) {
	x := model.Asset{}
	result := store.db.
		Where("case_id = ?", cid).
		First(&x, "name = ?", name)
	return x, result.Error
}

func (store *Store) SaveAsset(cid int64, x model.Asset) (model.Asset, error) {
	x.CRC = model.HashFields(
		x.CaseID,
		x.Type,
		x.Name,
		x.IP,
		x.Description,
		x.Compromised,
		x.Analysed,
	)

	result := store.db.
		Where("case_id = ?", cid).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "crc"}}, DoNothing: true}).
		Save(&x)
	return x, result.Error
}

func (store *Store) DeleteAsset(cid int64, id int64) error {
	x := model.Asset{}
	return store.db.
		Where("case_id = ?", cid).
		Delete(&x, id).Error
}
