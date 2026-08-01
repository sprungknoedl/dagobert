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

// Zircolite ships no vendored assets of its own inside the module's install tree that
// Dagobert can rely on (see docs/specs/update-dockerfile.md): the Timesketch export
// template, its default EVTX ruleset, and the config.yaml + transforms/ that -c requires.
// UpdateAssets fetches all four into external/zircolite/, mirroring chainsaw.go.
var (
	TemplateFile  = filepath.Join(model.VendorDir, "zircolite", "exportForTimesketch.tmpl")
	RulesetFile   = filepath.Join(model.VendorDir, "zircolite", "rules_windows_generic.json")
	ConfigFile    = filepath.Join(model.VendorDir, "zircolite", "config.yaml")
	TransformsDir = filepath.Join(model.VendorDir, "zircolite", "transforms")
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

	for _, asset := range []struct {
		label string
		path  string
	}{
		{"timesketch export template", TemplateFile},
		{"ruleset", RulesetFile},
		{"config", ConfigFile},
		{"transforms directory", TransformsDir},
	} {
		if _, err := os.Stat(asset.path); err != nil {
			err = fmt.Errorf("%s %q: %w", asset.label, asset.path, err)
			slog.Warn("validating module prerequisites failed", "module", "zircolite", "err", err)
			return nil, err
		}
	}

	return m, nil
}

// UpdateAssets fetches Zircolite's pinned external vendor data from
// wagga40/Zircolite's v3.7.6 tag into external/zircolite/. Pinned to a tag rather than
// tracking master (unlike Chainsaw's UpdateAssets): TemplateFile reads Zircolite's own
// result structure, so a drifting template against a pinned tool would break quietly.
func (m *Module) UpdateAssets(ctx context.Context) error {
	if os.Getenv("MODULE_ZIRCOLITE") == "" {
		slog.Info("module disabled, skipping asset fetch", "module", "zircolite")
		return nil
	}

	body, err := utils.DownloadZip(ctx, "https://github.com/wagga40/Zircolite/archive/refs/tags/v3.7.6.zip")
	if err != nil {
		return fmt.Errorf("fetching zircolite assets: %w", err)
	}

	const root = "Zircolite-3.7.6/"
	if err := utils.ExtractZipFile(body, root+"templates/exportForTimesketch.tmpl", TemplateFile); err != nil {
		return fmt.Errorf("extracting timesketch export template: %w", err)
	}
	if err := utils.ExtractZipFile(body, root+"rules/rules_windows_generic.json", RulesetFile); err != nil {
		return fmt.Errorf("extracting ruleset: %w", err)
	}
	// config.yaml and transforms/ are extracted as siblings (both directly under
	// external/zircolite/) so config.yaml's "transforms_dir:" resolves correctly.
	if err := utils.ExtractZipFile(body, root+"config/config.yaml", ConfigFile); err != nil {
		return fmt.Errorf("extracting config: %w", err)
	}
	if err := utils.ExtractZipSubtree(body, root+"config/transforms/", TransformsDir); err != nil {
		return fmt.Errorf("extracting transforms: %w", err)
	}

	return nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	evidence, err := utils.GuardEvidenceRun(m, job)
	if err != nil {
		return err
	}

	src := utils.Filepath(evidence)
	raw := src + ".zircolite.raw.json"
	dst := src + ".zircolite.jsonl"
	logfile := filepath.Join(model.TmpDir, filepath.Base(src)+".zircolite.log")

	if err := os.MkdirAll(model.TmpDir, 0755); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:],
		"--evtx", src,
		"-o", raw,
		"-c", ConfigFile,
		"-r", RulesetFile,
		"-t", TemplateFile,
		"-T", dst,
		"--logfile", logfile,
	)...)

	slog.Debug("running command", "module", "zircolite", "args", cmd.Args)
	// TODO: output is discarded on success; to persist it, capture it here and store it
	// somewhere on Job (no field for this today - would need a new column/migration or a
	// log file under files/) instead of dropping it.
	out, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
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
