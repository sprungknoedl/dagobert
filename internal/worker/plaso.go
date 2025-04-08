package worker

import (
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var argsPlaso []string

func ValidatePlaso() []string {
	var err error

	_, argsPlaso, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_PLASO"))
	if err != nil || len(argsPlaso) < 1 {
		slog.Warn("validating module prerequisites failed", "module", "plaso", "step", "shell parsing", "err", err)
		return nil
	}

	slog.Info("validating module prerequisites", "module", "plaso")
	cmd := exec.Command(argsPlaso[0], append(argsPlaso[1:], "-V")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("validating module prerequisites failed", "module", "plaso", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return nil
	}

	return []string{
		"Plaso (Windows Preset)",
		"Plaso (Linux Preset)",
		"Plaso (MacOS Preset)",
		"Plaso (Filesystem Timeline)"}
}

func runPlaso(job Job, parsers string, ext string) error {
	src := Filepath(job.Evidence)
	dst := src + ext

	cmd := exec.CommandContext(job.Ctx, argsPlaso[0], append(argsPlaso[1:],
		"--unattended",
		"--parsers", parsers,
		"--output-format", "dynamic",
		"--source", src,
		"--storage-file", dst,
		"--write", dst+".csv",
	)...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| running command: %s", tty.Cyan(" DEB "), cmd.Args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(dst)
		os.Remove(dst + ".csv")
		return err
	}

	if err := AddFromFS(model.Evidence{
		ID:     "new",
		CaseID: job.Case.ID,
		Type:   "Other",
		Name:   filepath.Base(dst),
		Source: job.Evidence.Source,
		Notes:  "module-plaso",
	}); err != nil {
		return err
	}

	if err := AddFromFS(model.Evidence{
		ID:     "new",
		CaseID: job.Case.ID,
		Type:   "Other",
		Name:   filepath.Base(dst) + ".csv",
		Source: job.Evidence.Source,
		Notes:  "module-plaso",
	}); err != nil {
		return err
	}

	return nil
}

func RunPlasoWindows(job Job) error {
	return runPlaso(job, "win7", ".plaso")
}

func RunPlasoLinux(job Job) error {
	return runPlaso(job, "linux", ".plaso")
}

func RunPlasoMacOS(job Job) error {
	return runPlaso(job, "macos", ".plaso")
}

func RunPlasoMFT(job Job) error {
	return runPlaso(job, "mft", ".mft.plaso")
}
