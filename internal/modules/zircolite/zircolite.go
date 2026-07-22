// Package zircolite implements the Zircolite evidence-processing module.
package zircolite

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
	return "Zircolite"
}

func (m *Module) Description() string {
	return "Zircolite is a standalone sigma-based detection tool for EVTX, Auditd, and Sysmon for Linux logs."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return filepath.Ext(e.Name) == ".evtx"
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	var err error
	_, m.args, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_ZIRCOLITE"))
	if err != nil {
		err = fmt.Errorf("invalid command in MODULE_ZIRCOLITE: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "zircolite", "err", err)
		return nil, err
	}
	if len(m.args) < 1 {
		slog.Info("module disabled, not configured", "module", "zircolite")
		return nil, errors.New("MODULE_ZIRCOLITE is not set, module disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("validating module prerequisites", "module", "zircolite")
	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:], "--version")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("command %q is not runnable: %w", m.args[0], err)
		slog.Warn("validating module prerequisites failed", "module", "zircolite", "err", err)
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
	raw := src + ".zircolite.raw.json"
	dst := src + ".zircolite.jsonl"

	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:],
		"--evtx", src,
		"-o", raw,
		"-t", "templates/exportForTimesketch.tmpl",
		"-T", dst,
	)...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Debug("running command", "module", "zircolite", "args", cmd.Args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		for _, f := range []string{dst, raw} {
			if rerr := os.Remove(f); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
				slog.Warn("failed to remove partial output file", "module", "zircolite", "err", rerr, "path", f)
			}
		}
		return err
	}

	// raw is Zircolite's mandatory -o output; it is pure clutter once the
	// templated -T output exists, so it's discarded unconditionally.
	if rerr := os.Remove(raw); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		slog.Warn("failed to remove raw output file", "module", "zircolite", "err", rerr, "path", raw)
	}

	if err := utils.AddFromFS(store, model.Evidence{
		CaseID: evidence.CaseID,
		Type:   "Logs",
		Name:   filepath.Base(dst),
		Source: evidence.Source,
		Notes:  "module-zircolite",
	}, m.Name()); err != nil {
		return err
	}

	return nil
}

func (m *Module) RenderSettings() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return nil })
}
