package timesketch

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
	return "Timesketch Importer"
}

func (m *Module) Description() string {
	return "Timesketch is an open-source tool for collaborative forensic timeline analysis. Using sketches you and your collaborators can organize and work together."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return strings.HasSuffix(e.Name, ".plaso") || strings.HasSuffix(e.Name, ".jsonl")
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	var err error
	_, m.args, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_TIMESKETCH"))
	if err != nil || len(m.args) < 1 {
		slog.Warn("validating module prerequisites failed", "module", "timesketch", "step", "shell parsing", "err", err)
		return nil, err
	}

	slog.Info("validating module prerequisites", "module", "timesketch")
	cmd := exec.Command(m.args[0], append(m.args[1:], "--version")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("validating module prerequisites failed", "module", "timesketch", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(job model.Job) error {
	evidence, ok := job.Object.Payload.(model.Evidence)
	if !ok {
		return fmt.Errorf("timesketch: unsupported type '%T'", job.Object.Payload)
	}

	src := workerutils.Filepath(evidence)
	cmd := exec.CommandContext(job.Ctx, m.args[0], append(m.args[1:],
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
