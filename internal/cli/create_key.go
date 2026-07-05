package cli

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/internal/model"
)

func CreateKey(cmd *cobra.Command, args []string) error {
	name := args[0]

	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	// Mint the key exactly like the UI does: persist only the hash + hint,
	// reveal the plaintext once. See handler/settings-keys.go Save.
	plaintext, hash, hint := model.GenerateKey()
	slog.Info("Adding key", "name", name)
	obj := model.Key{
		Key:  hash,
		Hint: hint,
		Name: name,
		Type: "API",
	}
	if err := db.SaveKey(obj); err != nil {
		slog.Error("failed to create key", "err", err)
		return err
	}

	// Print the plaintext to stdout (not the logger): the operator captures it
	// from their terminal here and now. It is never stored and cannot be
	// recovered later.
	fmt.Println(plaintext)
	return nil
}
