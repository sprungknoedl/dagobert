package model

import (
	"database/sql"
)

type User struct {
	ID    string
	Name  string
	UPN   string
	Email string
}

func (store *Store) FindUsers(search string, sort string) ([]User, error) {
	query := `
	SELECT id, name, upn, email
	FROM users
	WHERE 
		instr(name, :search) > 0 OR
		instr(upn, :search) > 0 OR
		instr(email, :search) > 0
	ORDER BY
		CASE WHEN :sort = 'upn'     THEN upn END ASC,
		CASE WHEN :sort = '-upn'    THEN upn END DESC,
		CASE WHEN :sort = 'email'      THEN email END ASC,
		CASE WHEN :sort = '-email'     THEN email END DESC,
		CASE WHEN :sort = '-name' THEN name END DESC,
		name ASC`

	rows, err := store.db.Query(query,
		sql.Named("search", search),
		sql.Named("sort", sort))
	if err != nil {
		return nil, err
	}

	var list []User
	err = ScanAll(rows, &list)
	return list, err
}

func (store *Store) GetUser(id string) (User, error) {
	query := `
	SELECT id, name, upn, email
	FROM users
	WHERE id = :id
	LIMIT 1`

	rows, err := store.db.Query(query,
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
	REPLACE INTO users (id, name, upn, email)
	VALUES (NULLIF(:id, ''), :name, :upn, :email)`

	_, err := store.db.Exec(query,
		sql.Named("id", obj.ID),
		sql.Named("name", obj.Name),
		sql.Named("upn", obj.UPN),
		sql.Named("email", obj.Email))
	return err
}

func (store *Store) DeleteUser(id string) error {
	query := `
	DELETE FROM users
	WHERE id = :id`

	_, err := store.db.Exec(query,
		sql.Named("id", id))
	return err
}
