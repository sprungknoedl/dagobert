// Package plaso implements the Plaso evidence-processing module.
package plaso

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules/utils"
)

var DefaultProfile = "win7"
var AllowedProfiles = []string{"win7", "linux", "macos", "mft"}

type Module struct {
	args []string
}

func NewModule() *Module {
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
	if err != nil {
		err = fmt.Errorf("invalid command in MODULE_PLASO: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "plaso", "err", err)
		return nil, err
	}
	if len(m.args) < 1 {
		slog.Info("module disabled, not configured", "module", "plaso")
		return nil, errors.New("MODULE_PLASO is not set, module disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("validating module prerequisites", "module", "plaso")
	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:], "-V")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("command %q is not runnable: %w", m.args[0], err)
		slog.Warn("validating module prerequisites failed", "module", "plaso", "err", err)
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	evidence, err := utils.GuardEvidenceRun(m, job)
	if err != nil {
		return err
	}

	src := utils.Filepath(evidence)
	dst := src + ".plaso"

	parser := cmp.Or(job.Settings["profile"], DefaultProfile)
	if !slices.Contains(AllowedProfiles, parser) {
		parser = DefaultProfile
	}

	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:],
		"--unattended",
		"--parsers", parser,
		"--output-format", "dynamic",
		"--source", src,
		"--storage-file", dst,
		"--write", dst+".csv",
	)...)

	slog.Debug("running command", "module", "plaso", "args", cmd.Args)
	// TODO: output is discarded on success; to persist it, capture it here and store it
	// somewhere on Job (no field for this today - would need a new column/migration or a
	// log file under files/) instead of dropping it.
	out, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
		// try to clean up
		for _, f := range []string{dst, dst + ".csv"} {
			if rerr := os.Remove(f); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
				slog.Warn("failed to remove partial output file", "module", "plaso", "err", rerr, "path", f)
			}
		}
		return err
	}

	return store.Transaction(func(tx *model.Store) error {
		if err := utils.AddFromFS(tx, model.Evidence{
			CaseID: evidence.CaseID,
			Type:   "Other",
			Name:   filepath.Base(dst),
			Source: evidence.Source,
			Notes:  "module-plaso",
		}, m.Name()); err != nil {
			return err
		}

		if err := utils.AddFromFS(tx, model.Evidence{
			CaseID: evidence.CaseID,
			Type:   "Other",
			Name:   filepath.Base(dst) + ".csv",
			Source: evidence.Source,
			Notes:  "module-plaso",
		}, m.Name()); err != nil {
			return err
		}
		return nil
	})
}
