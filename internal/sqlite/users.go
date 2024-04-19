package sqlite

import (
	"github.com/oklog/ulid/v2"
	"github.com/sprungknoedl/dagobert/pkg/model"
)

var _ model.UserStore = &Store{}

func (store *Store) ListUsers() ([]model.User, error) {
	var list []model.User
	result := store.db.
		Order("name asc").
		Find(&list)
	return list, result.Error
}

func (store *Store) FindUsers(search string, sort string) ([]model.User, error) {
	var list []model.User
	query := store.db.Where(store.db.
		Where("instr(name, ?) > 0", search).
		Or("instr(upn, ?) > 0", search).
		Or("instr(email, ?) > 0", search))

	switch sort {
	case "email":
		query = query.Order("email asc, name asc")
	case "-email":
		query = query.Order("email desc, name asc")
	case "upn":
		query = query.Order("upn asc, name asc")
	case "-upn":
		query = query.Order("upn desc, name asc")
	case "-name":
		query = query.Order("name desc")
	default: // case "name"
		query = query.Order("name asc")
	}

	result := query.Find(&list)
	return list, result.Error
}

func (store *Store) GetUser(id ulid.ULID) (model.User, error) {
	x := model.User{}
	result := store.db.
		First(&x, "id = ?", id)
	return x, result.Error
}

func (store *Store) SaveUser(x model.User) (model.User, error) {
	result := store.db.Save(&x)
	return x, result.Error
}

func (store *Store) DeleteUser(id ulid.ULID) error {
	x := model.User{}
	return store.db.
		Delete(&x, "id = ?", id).Error
}
