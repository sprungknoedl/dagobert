package worker

import (
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var argsTimesketch []string

func ValidateTimesketch() []string {
	var err error

	_, argsTimesketch, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_TIMESKETCH"))
	if err != nil || len(argsTimesketch) < 1 {
		slog.Warn("validating module prerequisites failed", "module", "timesketch", "step", "shell parsing", "err", err)
		return nil
	}

	slog.Info("validating module prerequisites", "module", "timesketch")
	cmd := exec.Command(argsTimesketch[0], append(argsTimesketch[1:], "--version")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("validating module prerequisites failed", "module", "timesketch", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return nil
	}

	return []string{"Timesketch Importer"}
}

func UploadToTimesketch(job Job) error {
	src := Filepath(job.Evidence)
	cmd := exec.CommandContext(job.Ctx, argsTimesketch[0], append(argsTimesketch[1:],
		"--quick",
		"--host", os.Getenv("TIMESKETCH_URL"),
		"-u", os.Getenv("TIMESKETCH_USER"),
		"-p", os.Getenv("TIMESKETCH_PASS"),
		"--sketch_id", strconv.Itoa(job.Case.SketchID),
		"--timeline-name", filepath.Base(src),
		"--context", "upload of dagobert evidence: "+filepath.Base(src),
		src,
	)...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| running command: %s", tty.Cyan(" DEB "), cmd.Args)
	err := cmd.Run()
	return err
}
