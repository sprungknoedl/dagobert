package plaso

import (
	"cmp"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker/workerutils"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var DefaultProfile = "win7"
var AllowedProfiles = []string{"win7", "linux", "macos", "mft"}

type Module struct {
	args []string
}

func NewModule() model.Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "Plaso"
}

func (m *Module) Description() string {
	return "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return filepath.Ext(e.Name) == ".zip"
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	var err error
	_, m.args, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_PLASO"))
	if err != nil || len(m.args) < 1 {
		slog.Warn("validating module prerequisites failed", "module", "plaso", "step", "shell parsing", "err", err)
		return nil, err
	}

	slog.Info("validating module prerequisites", "module", "plaso")
	cmd := exec.Command(m.args[0], append(m.args[1:], "-V")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("validating module prerequisites failed", "module", "plaso", "step", "cmd running", "err", err)
		os.Stderr.Write(out)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(job model.Job) error {
	evidence, ok := job.Object.Payload.(model.Evidence)
	if !ok {
		return fmt.Errorf("plaso: unsupported type '%T'", job.Object.Payload)
	}

	src := workerutils.Filepath(evidence)
	dst := src + ".plaso"

	parser := cmp.Or(job.Settings["profile"], DefaultProfile)
	if !slices.Contains(AllowedProfiles, parser) {
		parser = DefaultProfile
	}

	cmd := exec.CommandContext(job.Ctx, m.args[0], append(m.args[1:],
		"--unattended",
		"--parsers", parser,
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

	if err := workerutils.AddFromFS(model.Evidence{
		ID:     "new",
		CaseID: evidence.CaseID,
		Type:   "Other",
		Name:   filepath.Base(dst),
		Source: evidence.Source,
		Notes:  "module-plaso",
	}); err != nil {
		return err
	}

	if err := workerutils.AddFromFS(model.Evidence{
		ID:     "new",
		CaseID: evidence.CaseID,
		Type:   "Other",
		Name:   filepath.Base(dst) + ".csv",
		Source: evidence.Source,
		Notes:  "module-plaso",
	}); err != nil {
		return err
	}

	return nil
}
