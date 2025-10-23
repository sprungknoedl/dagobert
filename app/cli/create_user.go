package cli

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/aarondl/authboss/v3"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"golang.org/x/term"
)

func CreateUser(cmd *cobra.Command, args []string) error {
	id := fp.Random(32)
	username := args[0]

	// Connect to database
	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	ab, err := auth.Init(db)
	if err != nil {
		return err
	}

	// Collect password securely
	fmt.Print("Enter password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		slog.Error("failed to read password", "err", err)
		return err
	}

	slog.Info("Adding administrator", "uid", id, "upn", username)
	user := model.User{
		ID:   id,
		UPN:  username,
		Role: "Administrator",
	}
	db.Transaction(func(tx *model.Store) error {
		err = tx.SaveUser(user)
		if err != nil {
			return err
		}

		acl := auth.NewACL(tx)
		err = acl.SaveUserRole(id, "Administrator")
		if err != nil {
			return err
		}

		ab.Config.Storage.Server = tx
		err = ab.UpdatePassword(context.Background(), &user, string(password))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		slog.Error("failed to create user", "err", err)
	}

	return nil
}

func ChangePassword(cmd *cobra.Command, args []string) error {
	id := fp.Random(32)
	username := args[0]

	// Connect to database
	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	ab, err := auth.Init(db)
	if err != nil {
		return err
	}

	// Get user
	user, err := db.Load(cmd.Context(), username)
	if err != nil {
		return err
	}

	auser, ok := user.(authboss.AuthableUser)
	if !ok {
		return fmt.Errorf("unkown error: wrong user type")
	}

	// Collect password securely
	fmt.Print("Enter password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		slog.Error("failed to read password", "err", err)
		return err
	}

	slog.Info("Changing password for", "uid", id, "upn", username)
	db.Transaction(func(tx *model.Store) error {
		ab.Config.Storage.Server = tx
		err = ab.UpdatePassword(context.Background(), auser, string(password))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		slog.Error("failed to create user", "err", err)
	}

	return nil
}
