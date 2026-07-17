// Command dagobert is the entry point: it wires up the Cobra CLI (server, update, create-user, create-api-key, change-password).
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/internal/cli"
	"github.com/sprungknoedl/dagobert/internal/handler"
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

	cmd.Run = handler.Run // default command
	cmd.CompletionOptions = cobra.CompletionOptions{DisableDefaultCmd: true}
	cmd.AddCommand(&cobra.Command{Use: "server", Short: "Start web server and API.", Run: handler.Run})

	updateCmd := &cobra.Command{Use: "update", Short: "Create/upgrade the database and download MITRE ATT&CK data.", RunE: cli.Update}
	updateCmd.Flags().Bool("force", false, "Recover a dirty database (re-run the failed migration) and re-download the MITRE data.")
	cmd.AddCommand(updateCmd)

	cmd.AddCommand(&cobra.Command{Use: "create-user USERNAME", Short: "Create a user.", RunE: cli.CreateUser, Args: cobra.ExactArgs(1)})
	cmd.AddCommand(&cobra.Command{Use: "create-api-key NAME", Short: "Create an API key.", RunE: cli.CreateAPIKey, Args: cobra.ExactArgs(1)})
	cmd.AddCommand(&cobra.Command{Use: "change-password USERNAME", Short: "Change password for an user.", RunE: cli.ChangePassword, Args: cobra.ExactArgs(1)})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
