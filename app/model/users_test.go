package model

import (
	"context"
	"testing"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupDB() (*Store, func() error) {
	db, err := Connect(":memory:")
	if err != nil {
		panic(err)
	}

	// ignore errors here, as we would just panic ourself
	source, _ := iofs.New(Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	err = m.Up()
	if err != nil {
		panic(err)
	}

	return db, db.RawConn.Close
}

func TestGetUser(t *testing.T) {
	db, close := setupDB()
	defer close()

	t.Run("Normal User", func(t *testing.T) {
		user := User{ID: fp.Random(64), Name: "Max Musermann"}
		assert.Nil(t, db.SaveUser(user))

		user2, err := db.GetUser(user.ID)
		assert.Nil(t, err)
		assert.Equal(t, user, user2)
	})

	t.Run("Non-existant User", func(t *testing.T) {
		id := fp.Random(64)
		user, err := db.GetUser(id)
		assert.Zero(t, user)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestSaveUser(t *testing.T) {
	db, close := setupDB()
	defer close()

	t.Run("Normal User", func(t *testing.T) {
		user := User{ID: fp.Random(64), Name: "Max Musermann"}
		assert.Nil(t, db.SaveUser(user))
	})

	t.Run("Non-unique User", func(t *testing.T) {
		user1 := User{ID: fp.Random(64), Name: "Max Musermann", UPN: "max@mustermann.com"}
		user2 := User{ID: fp.Random(64), Name: "Max Musermann", UPN: "max@mustermann.com"}
		assert.Nil(t, db.SaveUser(user1))
		assert.Error(t, db.SaveUser(user2)) // should fail because UPN must be unique
	})

	t.Run("Empty User", func(t *testing.T) {
		user := User{}
		assert.Error(t, db.SaveUser(user))
	})
}

func TestSave(t *testing.T) {
	db, close := setupDB()
	defer close()

	t.Run("Normal User", func(t *testing.T) {
		user := User{ID: fp.Random(64), Name: "Max Musermann"}
		assert.Nil(t, db.SaveUser(user))
		assert.Nil(t, db.Save(context.Background(), &user))
	})

	t.Run("Non-existant User", func(t *testing.T) {
		err := db.Save(context.Background(), &User{
			ID:   "this user does not exist",
			Name: "this user does not exist",
		})

		assert.NotNil(t, err)
		assert.Equal(t, authboss.ErrUserNotFound, err)
	})
}
