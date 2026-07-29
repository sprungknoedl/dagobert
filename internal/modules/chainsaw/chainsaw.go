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
// Hayabusa (which bundles both). Rather than a MODULE_CHAINSAW_* env var per path
// (mirroring how mitre/ and files/ are fixed conventions elsewhere, not configurable), an
// operator who wants Chainsaw just places these at fixed paths relative to the working
// directory.
const (
	SigmaDir    = "sigma_rules"
	MappingFile = "mappings/sigma-event-logs-all.yml"
	RulesDir    = "rules"
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
		slog.Warn("validating module prerequisites failed", "module", "chainsaw", "err", err)
		return nil, err
	}
	if len(m.args) < 1 {
		slog.Info("module disabled, not configured", "module", "chainsaw")
		return nil, errors.New("MODULE_CHAINSAW is not set, module disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("validating module prerequisites", "module", "chainsaw")
	cmd := exec.CommandContext(ctx, m.args[0], append(m.args[1:], "--version")...)
	if out, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("command %q is not runnable: %w", m.args[0], err)
		slog.Warn("validating module prerequisites failed", "module", "chainsaw", "err", err)
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
		return nil, err
	}

	if _, err := os.Stat(SigmaDir); err != nil {
		err = fmt.Errorf("sigma rules directory %q: %w", SigmaDir, err)
		slog.Warn("validating module prerequisites failed", "module", "chainsaw", "err", err)
		return nil, err
	}

	if _, err := os.Stat(MappingFile); err != nil {
		err = fmt.Errorf("mapping file %q: %w", MappingFile, err)
		slog.Warn("validating module prerequisites failed", "module", "chainsaw", "err", err)
		return nil, err
	}

	// the native rules directory is optional - a missing path just omits -r in Run()
	if _, err := os.Stat(RulesDir); err == nil {
		m.hasRules = true
	} else {
		slog.Info("chainsaw native rules directory not found, omitting -r", "module", "chainsaw", "path", RulesDir)
	}

	return m, nil
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

	slog.Debug("running command", "module", "chainsaw", "args", cmd.Args)
	// TODO: output is discarded on success; to persist it, capture it here and store it
	// somewhere on Job (no field for this today - would need a new column/migration or a
	// log file under files/) instead of dropping it.
	out, err := cmd.CombinedOutput()
	if err != nil {
		_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
		// try to clean up
		for _, f := range []string{raw, dst} {
			if rerr := os.Remove(f); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
				slog.Warn("failed to remove partial output file", "module", "chainsaw", "err", rerr, "path", f)
			}
		}
		return err
	}

	if err := rewrite(raw, dst); err != nil {
		for _, f := range []string{raw, dst} {
			if rerr := os.Remove(f); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
				slog.Warn("failed to remove partial output file", "module", "chainsaw", "err", rerr, "path", f)
			}
		}
		return err
	}
	if rerr := os.Remove(raw); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		slog.Warn("failed to remove intermediate output file", "module", "chainsaw", "err", rerr, "path", raw)
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
			slog.Warn("failed to close raw chainsaw output", "module", "chainsaw", "err", cerr, "path", src)
		}
	}()

	fw, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fw.Close(); cerr != nil {
			slog.Warn("failed to close rewritten chainsaw output", "module", "chainsaw", "err", cerr, "path", dst)
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
