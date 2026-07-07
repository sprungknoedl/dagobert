package cli

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/internal/auth"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"golang.org/x/crypto/bcrypt"
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

	// Collect password securely
	fmt.Print("Enter password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		slog.Error("failed to read password", "err", err)
		return err
	}

	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "err", err)
		return err
	}

	slog.Info("Adding administrator", "uid", id, "upn", username)
	user := model.User{
		ID:       id,
		UPN:      username,
		Role:     "Administrator",
		Password: string(hash),
	}
	err = db.Transaction(func(tx *model.Store) error {
		if err := tx.SaveUser(user); err != nil {
			return err
		}

		acl := auth.NewACL(tx)
		return acl.SaveUserRole(id, "Administrator")
	})
	if err != nil {
		slog.Error("failed to create user", "err", err)
	}

	return nil
}

func ChangePassword(cmd *cobra.Command, args []string) error {
	username := args[0]

	// Connect to database
	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	// Get user
	user, err := db.GetUserByUPN(username)
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

	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "err", err)
		return err
	}

	slog.Info("Changing password for", "uid", user.ID, "upn", username)
	user.Password = string(hash)
	if err := db.SaveUser(user); err != nil {
		slog.Error("failed to change password", "err", err)
		return err
	}

	return nil
}
