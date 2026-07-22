// Package dissect implements the Dissect (dissect.target) evidence-processing module.
package dissect

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/mattn/go-shellwords"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules/utils"
)

// SupportedExtensions lists the file extensions dissect.target's loaders accept. Disk
// images and VM containers (E01/EWF, VMDK, VHD/VHDX, VDI, QCOW2, raw/dd, VMX/VMTX, VBOX,
// OVA/OVF, XVA, VBK) are opened via dissect.target's content-sniffing RawLoader, so these
// extensions are a Dagobert-side convention rather than something dissect.target checks
// itself; archive/triage-collection formats (zip, tar, ab, ad1, ufd/ufdx) are matched by
// dissect.target's own loaders by extension. Extension-only, no content sniffing, matching
// every other module's Supports().
var SupportedExtensions = []string{
	".e01", ".ex01", ".vmdk", ".vhd", ".vhdx", ".vdi", ".qcow2", ".raw", ".dd",
	".vmx", ".vmtx", ".vbox", ".ova", ".ovf", ".xva", ".vbk",
	".zip", ".tar", ".ab", ".ad1", ".ufd", ".ufdx",
}

var DefaultGroup = "mft"
var AllowedGroups = map[string]string{
	"mft":       "mft",
	"evtx":      "evtx",
	"artifacts": "runkeys,services,prefetch,shimcache,userassist,lnk",
}

type Module struct {
	args      []string
	rdumpArgs []string
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "Dissect"
}

func (m *Module) Description() string {
	return "Dissect (dissect.target) is a Python framework for forensic acquisition and analysis, purpose-built for fast, low-noise processing of live-collection triage bundles as well as disk images and VM containers."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return slices.Contains(SupportedExtensions, filepath.Ext(e.Name))
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	var err error
	_, m.args, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_DISSECT"))
	if err != nil {
		err = fmt.Errorf("invalid command in MODULE_DISSECT: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "dissect", "err", err)
		return nil, err
	}
	if len(m.args) < 1 {
		slog.Info("module disabled, not configured", "module", "dissect")
		return nil, errors.New("MODULE_DISSECT is not set, module disabled")
	}

	_, m.rdumpArgs, err = shellwords.ParseWithEnvs(os.Getenv("MODULE_DISSECT_RDUMP"))
	if err != nil {
		err = fmt.Errorf("invalid command in MODULE_DISSECT_RDUMP: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "dissect", "err", err)
		return nil, err
	}
	if len(m.rdumpArgs) < 1 {
		slog.Info("module disabled, not configured", "module", "dissect")
		return nil, errors.New("MODULE_DISSECT_RDUMP is not set, module disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("validating module prerequisites", "module", "dissect")
	for _, args := range [][]string{m.args, m.rdumpArgs} {
		cmd := exec.CommandContext(ctx, args[0], append(args[1:], "--version")...)
		if out, err := cmd.CombinedOutput(); err != nil {
			err = fmt.Errorf("command %q is not runnable: %w", args[0], err)
			slog.Warn("validating module prerequisites failed", "module", "dissect", "err", err)
			_, _ = os.Stderr.Write(out) //nolint:errcheck // best-effort diagnostic dump; err is already captured and returned
			return nil, err
		}
	}

	return m, nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	evidence, err := utils.GuardEvidenceRun(m, job)
	if err != nil {
		return err
	}

	src := utils.Filepath(evidence)
	raw := src + ".dissect.raw.jsonl"
	dst := src + ".dissect.jsonl"

	group := cmp.Or(job.Settings["profile"], DefaultGroup)
	functions, ok := AllowedGroups[group]
	if !ok {
		functions = AllowedGroups[DefaultGroup]
	}

	// target-query -f <funcs> <src> | rdump --multi-timestamp -w 'jsonfile://<raw>'
	// jsonfile:// (rather than -J/--jsonlines, which is shorthand for
	// jsonfile://?descriptors=false) is required to keep _recorddescriptor on each
	// record, which rewrite() below needs to pick a message template.
	query := exec.CommandContext(ctx, m.args[0], append(m.args[1:], "-f", functions, src)...)
	dump := exec.CommandContext(ctx, m.rdumpArgs[0], append(m.rdumpArgs[1:], "--multi-timestamp", "-w", "jsonfile://"+raw)...)

	stdout, err := query.StdoutPipe()
	if err != nil {
		return err
	}
	dump.Stdin = stdout
	query.Stderr = os.Stderr
	dump.Stdout = os.Stdout
	dump.Stderr = os.Stderr

	slog.Debug("running command", "module", "dissect", "args", query.Args)
	if err := query.Start(); err != nil {
		return err
	}

	slog.Debug("running command", "module", "dissect", "args", dump.Args)
	if err := dump.Start(); err != nil {
		// unblock and reap the already-started producer before returning
		_ = query.Process.Kill()
		_ = query.Wait()
		return err
	}

	// dump is the reader of query's stdout pipe, so it must be waited on before
	// query - see the os/exec.Cmd.StdoutPipe docs.
	dumpErr := dump.Wait()
	queryErr := query.Wait()
	if err := cmp.Or(queryErr, dumpErr); err != nil {
		// try to clean up
		if rerr := os.Remove(raw); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			slog.Warn("failed to remove partial output file", "module", "dissect", "err", rerr, "path", raw)
		}
		return err
	}

	if err := rewrite(raw, dst); err != nil {
		if rerr := os.Remove(dst); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			slog.Warn("failed to remove partial output file", "module", "dissect", "err", rerr, "path", dst)
		}
		return err
	}
	if rerr := os.Remove(raw); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		slog.Warn("failed to remove intermediate output file", "module", "dissect", "err", rerr, "path", raw)
	}

	return utils.AddFromFS(store, model.Evidence{
		CaseID: evidence.CaseID,
		Type:   "Logs",
		Name:   filepath.Base(dst),
		Source: evidence.Source,
		Notes:  "module-dissect",
	})
}

// rewrite reads target-query/rdump's raw jsonlines output at src, skipping
// "recorddescriptor" lines (rdump emits one per unique record shape encountered,
// interspersed throughout the file rather than grouped at the top), and writes dst with
// Timesketch's required message/datetime/timestamp_desc fields added to each record.
// ts/ts_description are renamed to datetime/timestamp_desc rather than duplicated; every
// other original field is kept as-is.
func rewrite(src, dst string) error {
	fr, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fr.Close(); cerr != nil {
			slog.Warn("failed to close raw dissect output", "module", "dissect", "err", cerr, "path", src)
		}
	}()

	fw, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fw.Close(); cerr != nil {
			slog.Warn("failed to close rewritten dissect output", "module", "dissect", "err", cerr, "path", dst)
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

func rewriteLine(line []byte, w io.Writer) error {
	var rec map[string]any
	if err := json.Unmarshal(line, &rec); err != nil {
		return err
	}

	if rec["_type"] == "recorddescriptor" {
		return nil
	}

	// compute the message before renaming ts/ts_description, since the message
	// templates and messageFallback both read them under their original names
	rec["message"] = messageFor(recordDescriptor(rec), rec)
	rec["datetime"] = rec["ts"]
	rec["timestamp_desc"] = rec["ts_description"]
	delete(rec, "ts")
	delete(rec, "ts_description")

	enc, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	enc = append(enc, '\n')
	_, err = w.Write(enc)
	return err
}

// recordDescriptor extracts a record's schema name from its _recorddescriptor field,
// e.g. ["filesystem/ntfs/mft/std", 3050922189] -> "filesystem/ntfs/mft/std", ignoring the
// trailing schema hash.
func recordDescriptor(rec map[string]any) string {
	arr, ok := rec["_recorddescriptor"].([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	name, _ := arr[0].(string)
	return name
}

// messageTemplates renders a human-readable Timesketch message per record descriptor
// name. Descriptors without an entry fall back to messageFallback.
var messageTemplates = map[string]func(rec map[string]any) string{
	// mft: filesystem/ntfs/mft/std uses its own "ts_type" field (B/C/M/A) to
	// distinguish the four $STANDARD_INFORMATION timestamps of one MFT entry -
	// "ts_description" is always the literal string "ts" here (rdump's multi-timestamp
	// auto-label for a field that's already just called "ts"), not the B/C/M/A code.
	"filesystem/ntfs/mft/std": func(rec map[string]any) string {
		return fmt.Sprintf("%v was %v ($STANDARD_INFORMATION)", rec["path"], mftAction(rec["ts_type"]))
	},
	// mft also emits this second shape for the $FILE_NAME attribute - worth calling
	// out separately from $STANDARD_INFORMATION since it's less commonly timestomped,
	// useful for cross-checking. Same ts_type caveat as std.
	"filesystem/ntfs/mft/filename": func(rec map[string]any) string {
		return fmt.Sprintf("%v was %v ($FILE_NAME)", rec["path"], mftAction(rec["ts_type"]))
	},
	"filesystem/windows/evtx": func(rec map[string]any) string {
		return fmt.Sprintf("Event %v logged by %v (%v)", rec["EventID"], rec["Provider_Name"], rec["Channel"])
	},
	// runkeys: ts is the registry key's last-write time, not a creation time, so
	// "last modified" rather than "created"/"installed". "command" is a nested
	// {executable, args} object.
	"windows/registry/run": func(rec map[string]any) string {
		return fmt.Sprintf("Autorun entry %q launches %v (registry key last modified)", rec["name"], commandString(rec))
	},
	// services: same "last-write time, not creation" caveat as runkeys.
	"windows/service": func(rec map[string]any) string {
		return fmt.Sprintf("Service %q (%v) registry entry last modified", rec["name"], rec["displayname"])
	},
	// prefetch: ts is one of potentially several recorded run timestamps (not
	// necessarily the most recent), and runcount is a running total rather than being
	// specific to this timestamp - phrasing avoids implying otherwise.
	"filesystem/ntfs/prefetch": func(rec map[string]any) string {
		return fmt.Sprintf("%v was executed (%v total runs recorded)", rec["filename"], rec["runcount"])
	},
	// shimcache: ts is the cached file's own last-modified time at caching time, not
	// an execution timestamp - ShimCache presence alone is not proof of execution, a
	// well-known DFIR pitfall worth spelling out in the message itself.
	"windows/shimcache": func(rec map[string]any) string {
		return fmt.Sprintf("%v referenced in ShimCache (file last modified; presence alone does not confirm execution)", rec["path"])
	},
	// userassist: unlike prefetch/shimcache, UserAssist's ts is documented as the
	// last-execution time, so "executed" is accurate here.
	"windows/registry/userassist": func(rec map[string]any) string {
		return fmt.Sprintf("%v was last executed (%v times total)", rec["path"], rec["number_of_executions"])
	},
	// lnk: ts_description is one of lnk_mtime/lnk_atime/lnk_ctime (the .lnk file's own
	// timestamps) or target_mtime/target_atime/target_ctime (the target file's
	// timestamps as snapshotted inside the shortcut, not live) - both map to the same
	// modified/accessed/created wording here.
	"windows/filesystem/lnk": func(rec map[string]any) string {
		return fmt.Sprintf("Shortcut %v was %v", rec["lnk_path"], lnkAction(rec["ts_description"]))
	},
}

// mftAction maps an MFT record's ts_type (B/C/M/A) to the timestamp it represents, per
// dissect.target's filesystem/ntfs/mft.py: B=creation_time, C=last_change_time,
// M=last_modification_time, A=last_access_time.
func mftAction(tsType any) string {
	switch fmt.Sprint(tsType) {
	case "B":
		return "created"
	case "C":
		return "metadata changed"
	case "M":
		return "modified"
	case "A":
		return "accessed"
	default:
		return fmt.Sprint(tsType)
	}
}

// lnkAction maps an lnk record's ts_description to the timestamp it represents; the
// lnk_/target_ prefix (see the windows/filesystem/lnk template) distinguishes which file
// it's about, so only the mtime/atime/ctime suffix matters here.
func lnkAction(tsDescription any) string {
	switch fmt.Sprint(tsDescription) {
	case "lnk_mtime", "target_mtime":
		return "modified"
	case "lnk_atime", "target_atime":
		return "accessed"
	case "lnk_ctime", "target_ctime":
		return "created"
	default:
		return fmt.Sprint(tsDescription)
	}
}

// commandString renders a runkeys record's nested "command" object
// ({"executable": ..., "args": [...]}, or null if the run key's value was empty) as a
// single command line.
func commandString(rec map[string]any) string {
	cmd, ok := rec["command"].(map[string]any)
	if !ok || cmd == nil {
		return "(no command)"
	}

	parts := []string{fmt.Sprint(cmd["executable"])}
	if args, ok := cmd["args"].([]any); ok {
		for _, arg := range args {
			parts = append(parts, fmt.Sprint(arg))
		}
	}
	return strings.Join(parts, " ")
}

// recordMetaFields are excluded from messageFallback: _-prefixed fields are
// rdump/flow.record bookkeeping, and ts/ts_description are already promoted onto the
// record as datetime/timestamp_desc.
var recordMetaFields = map[string]bool{
	"_type": true, "_recorddescriptor": true, "_source": true,
	"_classification": true, "_generated": true, "_version": true,
	"ts": true, "ts_description": true,
}

func messageFor(name string, rec map[string]any) string {
	if tmpl, ok := messageTemplates[name]; ok {
		return tmpl(rec)
	}
	return messageFallback(rec)
}

// messageFallback flattens a record's non-meta fields into a "key=value, ..." message
// for descriptor names without a dedicated template.
func messageFallback(rec map[string]any) string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		if !recordMetaFields[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, rec[k]))
	}
	return strings.Join(parts, ", ")
}
