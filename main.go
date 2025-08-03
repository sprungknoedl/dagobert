package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/cli"
	"github.com/sprungknoedl/dagobert/app/handler"
	"github.com/sprungknoedl/dagobert/app/worker"
)

type Configuration struct {
	AssetsFolder   string
	EvidenceFolder string

	Database string

	ClientId      string
	ClientSecret  string
	ClientUrl     string
	Issuer        string
	IdentityClaim string

	SessionSecret string
}

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.DateTime,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// write all errors in red
				if err, ok := a.Value.Any().(error); ok {
					aErr := tint.Err(err)
					aErr.Key = a.Key
					return aErr
				}
				return a
			},
		}),
	))

	cmd := &cobra.Command{
		Use:   "dagobert",
		Short: "Collaborative Incident Response Platform",
	}

	_ = cobra.Command{
		Args: cobra.ExactArgs(1),
	}

	cmd.Run = handler.Run // default command
	cmd.CompletionOptions = cobra.CompletionOptions{DisableDefaultCmd: true}
	cmd.AddCommand(&cobra.Command{Use: "server", Short: "Start web server and API.", Run: handler.Run})
	cmd.AddCommand(&cobra.Command{Use: "worker", Short: "Start background job worker.", Run: worker.Run})
	cmd.AddCommand(&cobra.Command{Use: "db", Short: "Perform database migrations.", RunE: cli.Migrate})
	cmd.AddCommand(&cobra.Command{Use: "create-user USERNAME", Short: "Create a user.", RunE: cli.CreateUser, Args: cobra.ExactArgs(1)})
	cmd.AddCommand(&cobra.Command{Use: "create-key NAME", Short: "Create a API key.", RunE: cli.CreateKey, Args: cobra.ExactArgs(1)})
	cmd.Execute()
}
