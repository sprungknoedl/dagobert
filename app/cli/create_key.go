package cli

import (
	"cmp"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func CreateKey(cmd *cobra.Command, args []string) error {
	key := fp.Random(64)
	name := args[0]

	dburl := cmp.Or(os.Getenv("DB_URL"), model.DefaultUrl)
	slog.Info("Connecting to database", "url", dburl)
	db, err := model.Connect(dburl)
	if err != nil {
		return err
	}

	slog.Info("Adding key", "name", name, "key", key)
	obj := model.Key{
		Key:  key,
		Name: name,
		Type: "API",
	}
	err = db.SaveKey(obj)
	if err != nil {
		slog.Error("failed to create key", "err", err)
	}

	return nil
}
