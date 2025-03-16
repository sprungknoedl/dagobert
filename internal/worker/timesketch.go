package worker

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strconv"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var argsTimesketch []string

func ValidateTimesketch() bool {
	var err error

	_, argsTimesketch, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_TIMESKETCH"))
	if err != nil || len(argsTimesketch) < 1 {
		slog.Error("validating module prerequisites failed", "module", "timesketch", "step", "shell parsing", "err", err)
		return false
	}

	slog.Info("validating module prerequisites", "module", "timesketch")
	cmd := exec.Command(argsTimesketch[0], append(argsTimesketch[1:], "--version")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Error("validating module prerequisites failed", "module", "timesketch", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return false
	}

	modules = append(modules, "Timesketch Importer")
	return true
}

func UploadToTimesketch(ctx context.Context, job Job) error {
	src := Filepath(job.Evidence)
	cmd := exec.CommandContext(ctx, argsTimesketch[0], append(argsTimesketch[1:],
		"--quick",
		"--host", os.Getenv("TIMESKETCH_URL"),
		"-u", os.Getenv("TIMESKETCH_USER"),
		"-p", os.Getenv("TIMESKETCH_PASS"),
		"--sketch_id", strconv.Itoa(job.Case.SketchID),
		src,
	)...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| running command: %s", tty.Cyan(" DEB "), cmd.Args)
	err := cmd.Run()
	return err
}
