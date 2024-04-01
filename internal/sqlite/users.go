package sqlite

import (
	"github.com/sprungknoedl/dagobert/model"
	"gorm.io/gorm/clause"
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

func (store *Store) GetUser(id int64) (model.User, error) {
	x := model.User{}
	result := store.db.
		First(&x, id)
	return x, result.Error
}

func (store *Store) SaveUser(x model.User) (model.User, error) {
	result := store.db.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.Set{
				clause.Assignment{Column: clause.Column{Name: "name"}, Value: x.Name},
				clause.Assignment{Column: clause.Column{Name: "upn"}, Value: x.UPN},
				clause.Assignment{Column: clause.Column{Name: "email"}, Value: x.Email},
			},
		}).
		Save(&x)
	return x, result.Error
}

func (store *Store) DeleteUser(id int64) error {
	x := model.User{}
	return store.db.
		Delete(&x, id).Error
}
