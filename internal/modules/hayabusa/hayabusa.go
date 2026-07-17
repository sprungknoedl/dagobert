// Package hayabusa implements the Hayabusa evidence-processing module.
package hayabusa

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/a-h/templ"
	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules/utils"
)

type Module struct {
	args []string
}

func NewModule() *Module {
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
	if err != nil {
		err = fmt.Errorf("invalid command in MODULE_HAYABUSA: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "hayabusa", "err", err)
		return nil, err
	}
	if len(m.args) < 1 {
		slog.Info("module disabled, not configured", "module", "hayabusa")
		return nil, errors.New("MODULE_HAYABUSA is not set, module disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("validating module prerequisites", "module", "hayabusa")
	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:], "help")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("command %q is not runnable: %w", m.args[0], err)
		slog.Warn("validating module prerequisites failed", "module", "hayabusa", "err", err)
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
	dst := src + ".jsonl"

	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:],
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

	slog.Debug("running command", "module", "hayabusa", "args", cmd.Args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		if rerr := os.Remove(dst); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			slog.Warn("failed to remove partial output file", "module", "hayabusa", "err", rerr, "path", dst)
		}
		return err
	}

	if err := utils.AddFromFS(store, model.Evidence{
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

func (m *Module) RenderSettings() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return nil })
}
