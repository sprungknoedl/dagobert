package model

import (
	"fmt"
)

var UserRoles = []string{"Administrator", "User", "Read-Only"}

type User struct {
	ID        string
	Name      string
	UPN       string
	Email     string
	Role      string
	LastLogin Time
}

func (u User) String() string {
	return fmt.Sprintf("%s (%s)", u.Name, u.UPN)
}

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

func (store *Store) SaveUser(obj User) error {
	return store.DB.Save(obj).Error
}

func (store *Store) DeleteUser(id string) error {
	return store.DB.Delete(&User{}, "id = ?", id).Error
}
