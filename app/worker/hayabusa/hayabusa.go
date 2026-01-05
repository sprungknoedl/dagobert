package hayabusa

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker/workerutils"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

type Module struct {
	args []string
}

func NewModule() model.Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "Hayabusa"
}

func (m *Module) Description() string {
	return "Hayabusa (隼) is a sigma-based threat hunting and fast forensics timeline generator for Windows event logs."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return filepath.Ext(e.Name) == ".evtx"
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	var err error
	_, m.args, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_HAYABUSA"))
	if err != nil || len(m.args) < 1 {
		slog.Warn("validating hayabusa prerequisites failed", "step", "shell parsing", "err", err)
		return nil, fmt.Errorf("validating hayabusa prerequisites failed: %w", err)
	}

	slog.Info("validating module prerequisites", "module", "hayabusa")
	cmd := exec.Command(m.args[0], append(m.args[1:], "help")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("validating hayabusa prerequisites failed", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return nil, fmt.Errorf("validating hayabusa prerequisites failed: %w", err)
	}

	return m, nil
}

func (m *Module) Run(job model.Job) error {
	evidence, ok := job.Object.Payload.(model.Evidence)
	if !ok {
		return fmt.Errorf("hayabusa: unsupported type '%T'", job.Object.Payload)
	}

	src := workerutils.Filepath(evidence)
	dst := src + ".jsonl"

	cmd := exec.CommandContext(job.Ctx, m.args[0], append(m.args[1:],
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

	if err := workerutils.AddFromFS(model.Evidence{
		ID:     "new",
		CaseID: evidence.CaseID,
		Type:   "Logs",
		Name:   filepath.Base(dst),
		Source: evidence.Source,
		Notes:  "module-hayabusa",
	}); err != nil {
		return err
	}

	return nil
}
