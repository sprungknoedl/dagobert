// Package chainsaw implements the Chainsaw evidence-processing module.
package chainsaw

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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

// Chainsaw ships no Sigma rules, EVTX field mapping, or native rules of its own, unlike
// Hayabusa (which bundles both). UpdateAssets fetches them into external/chainsaw/ via
// `dagobert update`, mirroring how external/mitre/ is populated.
var (
	SigmaDir    = filepath.Join(model.VendorDir, "chainsaw", "sigma_rules")
	MappingFile = filepath.Join(model.VendorDir, "chainsaw", "mappings", "sigma-event-logs-all.yml")
	RulesDir    = filepath.Join(model.VendorDir, "chainsaw", "rules")
)

type Module struct {
	args     []string
	hasRules bool
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "Chainsaw"
}

func (m *Module) Description() string {
	return "Chainsaw is a Sigma-based and native-rule threat hunting tool for Windows Event Logs."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return filepath.Ext(e.Name) == ".evtx"
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	var err error
	_, m.args, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_CHAINSAW"))
	if err != nil {
		err = fmt.Errorf("invalid command in MODULE_CHAINSAW: %w", err)
		slog.Warn("validating module prerequisites failed", "module", m.Name(), "err", err)
		return nil, err
	}
	if len(m.args) < 1 {
		slog.Info("module disabled, not configured", "module", m.Name())
		return nil, errors.New("MODULE_CHAINSAW is not set, module disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("validating module prerequisites", "module", m.Name())
	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:], "--version")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("command %q is not runnable: %w", m.args[0], err)
		slog.Warn("validating module prerequisites failed", "module", m.Name(), "err", err)
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
		return nil, err
	}

	if _, err := os.Stat(SigmaDir); err != nil {
		err = fmt.Errorf("sigma rules directory %q: %w", SigmaDir, err)
		slog.Warn("validating module prerequisites failed", "module", m.Name(), "err", err)
		return nil, err
	}

	if _, err := os.Stat(MappingFile); err != nil {
		err = fmt.Errorf("mapping file %q: %w", MappingFile, err)
		slog.Warn("validating module prerequisites failed", "module", m.Name(), "err", err)
		return nil, err
	}

	// the native rules directory is optional - a missing path just omits -r in Run()
	if _, err := os.Stat(RulesDir); err == nil {
		m.hasRules = true
	} else {
		slog.Info("chainsaw native rules directory not found, omitting -r", "module", m.Name(), "path", RulesDir)
	}

	return m, nil
}

// UpdateAssets fetches Chainsaw's pinned external vendor data: SigmaHQ/sigma's
// rules into SigmaDir, and WithSecureLabs/chainsaw's mappings/rules into
// MappingFile's directory and RulesDir. Unlike MITRE's pinned release, these
// always fetch the master branch and overwrite unconditionally — no version
// pinning, no staleness check, no retries.
func (m *Module) UpdateAssets(ctx context.Context) error {
	if os.Getenv("MODULE_CHAINSAW") == "" {
		slog.Info("module disabled, skipping asset fetch", "module", m.Name())
		return nil
	}

	sigmaZip, err := utils.DownloadZip(ctx, "https://github.com/SigmaHQ/sigma/archive/refs/heads/master.zip")
	if err != nil {
		return fmt.Errorf("fetching sigma rules: %w", err)
	}
	if err := utils.ExtractZipSubtree(sigmaZip, "sigma-master/rules/", SigmaDir); err != nil {
		return fmt.Errorf("extracting sigma rules: %w", err)
	}

	chainsawZip, err := utils.DownloadZip(ctx, "https://github.com/WithSecureLabs/chainsaw/archive/refs/heads/master.zip")
	if err != nil {
		return fmt.Errorf("fetching chainsaw mappings/rules: %w", err)
	}
	if err := utils.ExtractZipSubtree(chainsawZip, "chainsaw-master/mappings/", filepath.Dir(MappingFile)); err != nil {
		return fmt.Errorf("extracting chainsaw mappings: %w", err)
	}
	if err := utils.ExtractZipSubtree(chainsawZip, "chainsaw-master/rules/", RulesDir); err != nil {
		return fmt.Errorf("extracting chainsaw rules: %w", err)
	}

	return nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	evidence, err := utils.GuardEvidenceRun(m, job)
	if err != nil {
		return err
	}

	src := utils.Filepath(evidence)
	raw := src + ".chainsaw.raw.jsonl"
	dst := src + ".chainsaw.jsonl"

	args := append(m.args[1:], "hunt", "-s", SigmaDir, "-m", MappingFile)
	if m.hasRules {
		args = append(args, "-r", RulesDir)
	}
	args = append(args, "--jsonl", "-o", raw, src)

	cmd := exec.CommandContext(ctx, m.args[0], args...)

	slog.Debug("running command", "module", m.Name(), "args", cmd.Args)
	// TODO: output is discarded on success; to persist it, capture it here and store it
	// somewhere on Job (no field for this today - would need a new column/migration or a
	// log file under data/) instead of dropping it.
	out, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
		// try to clean up
		for _, f := range []string{raw, dst} {
			if rerr := os.Remove(f); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
				slog.Warn("failed to remove partial output file", "module", m.Name(), "err", rerr, "path", f)
			}
		}
		return err
	}

	if err := rewrite(raw, dst); err != nil {
		for _, f := range []string{raw, dst} {
			if rerr := os.Remove(f); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
				slog.Warn("failed to remove partial output file", "module", m.Name(), "err", rerr, "path", f)
			}
		}
		return err
	}
	if rerr := os.Remove(raw); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		slog.Warn("failed to remove intermediate output file", "module", m.Name(), "err", rerr, "path", raw)
	}

	return utils.AddFromFS(store, model.Evidence{
		CaseID: evidence.CaseID,
		Type:   "Logs",
		Name:   filepath.Base(dst),
		Source: evidence.Source,
		Notes:  "module-chainsaw",
	}, m.Name())
}

func (m *Module) RenderSettings() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return nil })
}

// rewrite reads chainsaw hunt's `--jsonl` output at src - one hit record per line, each
// already carrying its rule's name/level/id/logsource inline (chainsaw resolves rule
// metadata onto the hit itself, unlike the raw Hit/Kind/Document structs in its Rust
// source) - and writes dst with Timesketch's required message/datetime/timestamp_desc
// fields added to each hit. Every other original field (including the matched event under
// "document"/"documents") is kept as-is.
func rewrite(src, dst string) error {
	fr, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fr.Close(); cerr != nil {
			slog.Warn("failed to close raw chainsaw output", "module", "Chainsaw", "err", cerr, "path", src)
		}
	}()

	fw, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fw.Close(); cerr != nil {
			slog.Warn("failed to close rewritten chainsaw output", "module", "Chainsaw", "err", cerr, "path", dst)
		}
	}()

	reader := bufio.NewReaderSize(fr, 64*1024)
	writer := bufio.NewWriter(fw)

	for {
		line, rerr := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if werr := rewriteLine(line, writer); werr != nil {
				return werr
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}

	return writer.Flush()
}

// rewriteLine renames a hit's "timestamp" field to "datetime" and adds "message"/
// "timestamp_desc", mirroring the message/datetime/timestamp_desc convention Zircolite's own
// Timesketch export template uses (message: rule name, timestamp_desc: "Event time").
func rewriteLine(line []byte, w io.Writer) error {
	var hit map[string]any
	if err := json.Unmarshal(line, &hit); err != nil {
		return err
	}

	hit["message"] = fmt.Sprintf("%v", hit["name"])
	hit["datetime"] = hit["timestamp"]
	hit["timestamp_desc"] = "Event time"
	delete(hit, "timestamp")

	enc, err := json.Marshal(hit)
	if err != nil {
		return err
	}
	enc = append(enc, '\n')
	_, err = w.Write(enc)
	return err
}
