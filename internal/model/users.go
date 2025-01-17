package model

import (
	"database/sql"
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
	query := `
	SELECT id, name, upn, email, role, last_login
	FROM users
	ORDER BY name ASC`

	rows, err := store.DB.Query(query)
	if err != nil {
		return nil, err
	}

	var list []User
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetUser(id string) (User, error) {
	query := `
	SELECT id, name, upn, email, role, last_login
	FROM users
	WHERE id = :id
	LIMIT 1`

	rows, err := store.DB.Query(query,
		sql.Named("id", id))
	if err != nil {
		return User{}, err
	}

	var obj User
	err = ScanOne(rows, &obj)
	return obj, err
}

func (store *Store) SaveUser(obj User) error {
	query := `
	INSERT INTO users (id, name, upn, email, role, last_login)
	VALUES (:id, :name, :upn, :email, :role, :last_login)
	ON CONFLICT (id)
		DO UPDATE SET name=:name, upn=:upn, email=:email, role=:role, last_login=:last_login
		WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("name", obj.Name),
		sql.Named("upn", obj.UPN),
		sql.Named("email", obj.Email),
		sql.Named("role", obj.Role),
		sql.Named("last_login", obj.LastLogin))
	return err
}

func (store *Store) DeleteUser(id string) error {
	query := `
	DELETE FROM users
	WHERE id = :id`

	_, err := store.DB.Exec(query,
		sql.Named("id", id))
	return err
}
