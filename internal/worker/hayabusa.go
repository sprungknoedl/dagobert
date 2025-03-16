package worker

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var argsHayabusa []string

func ValidateHayabusa() bool {
	var err error

	_, argsHayabusa, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_HAYABUSA"))
	if err != nil || len(argsHayabusa) < 1 {
		slog.Error("validating module prerequisites failed", "module", "hayabusa", "step", "shell parsing", "err", err)
		return false
	}

	slog.Info("validating module prerequisites", "module", "hayabusa")
	cmd := exec.Command(argsHayabusa[0], append(argsHayabusa[1:], "help")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Error("validating module prerequisites failed", "module", "hayabusa", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return false
	}

	modules = append(modules, "Hayabusa")
	return true
}

func RunHayabusa(ctx context.Context, job Job) error {
	src := Filepath(job.Evidence)
	dst := src + ".jsonl"

	cmd := exec.CommandContext(ctx, argsHayabusa[0], append(argsHayabusa[1:],
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--file", src,
		"--output", dst,
	)...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| running command: %s", tty.Cyan(" DEB "), cmd.Args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(dst)
		return err
	}

	if err := AddFromFS(model.Evidence{
		ID:     "new",
		CaseID: job.Case.ID,
		Type:   "Logs",
		Name:   filepath.Base(dst),
		Source: job.Evidence.Source,
		Notes:  "module-hayabusa",
	}); err != nil {
		return err
	}

	return nil
}
