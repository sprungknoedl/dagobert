package model

import (
	"errors"
	"fmt"
)

// var UserRoles = []string{"Administrator", "User", "Read-Only"}
var SystemUser = User{
	ID:   "<system>",
	Name: "System",
	Role: "Administrator",
}

var DonaldUser = User{
	ID:   "<donald>",
	Name: "Donald",
	Role: "Donald",
}

var ErrUserProtected = errors.New("modification to user prohibited")

type User struct {
	ID        string
	Name      string
	UPN       string
	Password  string
	Email     string
	Role      string
	LastLogin Time
}

func (u *User) String() string { return fmt.Sprintf("%s (%s)", u.Name, u.UPN) }

func (store *Store) ListUsers() ([]User, error) {
	list := []User{}
	tx := store.DB.
		Order("name asc").
		Find(&list)
	return list, tx.Error
}

func (store *Store) GetUser(id string) (User, error) {
	obj := User{}
	tx := store.DB.First(&obj, "id = ?", id)
	return obj, tx.Error
}

func (store *Store) GetUserByUPN(upn string) (User, error) {
	obj := User{}
	tx := store.DB.First(&obj, "upn = ?", upn)
	return obj, tx.Error
}

func (store *Store) SaveUser(obj User) error {
	if obj.ID == SystemUser.ID || obj.ID == DonaldUser.ID {
		return ErrUserProtected
	}
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteUser(id string) error {
	if id == SystemUser.ID || id == DonaldUser.ID {
		return ErrUserProtected
	}
	return store.DB.Delete(&User{}, "id = ?", id).Error
}
