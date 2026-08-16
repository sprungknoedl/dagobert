// Package alerts implements the Alert Importer module, which turns Sigma-derived,
// Timesketch-shaped JSONL alert files (as produced by Hayabusa, Zircolite, and
// Chainsaw) directly into Event rows on the case timeline.
package alerts

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules/utils"
)

var DefaultSeverity = "all"
var AllowedSeverities = []string{"all", "high", "critical"}

// techniqueRe matches a tag entry that names a single MITRE ATT&CK technique,
// with or without the "attack." prefix Sigma tags conventionally use.
var techniqueRe = regexp.MustCompile(`(?i)^(?:attack\.)?(t\d{4}(?:\.\d{3})?)$`)

type Module struct{}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "Alert Importer"
}

func (m *Module) Description() string {
	return "Alert Importer reads Sigma-derived, Timesketch-shaped JSONL alert files (as produced by Hayabusa, Zircolite, and Chainsaw) directly into the case timeline."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return strings.HasSuffix(e.Name, ".jsonl")
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	return m, nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job *model.Job) error {
	evidence, err := utils.GuardEvidenceRun(m, job)
	if err != nil {
		return err
	}

	severity := job.Settings["severity"]
	if !slices.Contains(AllowedSeverities, severity) {
		severity = DefaultSeverity
	}

	fr, err := os.Open(utils.Filepath(evidence))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fr.Close(); cerr != nil {
			slog.Warn("failed to close evidence file", "module", m.Name(), "err", cerr)
		}
	}()

	imported, filtered, skipped, err := importAlerts(store, evidence.CaseID, evidence.Name, fr, severity)
	if err != nil {
		return err
	}

	job.Results = map[string]string{
		"imported": strconv.Itoa(imported),
		"filtered": strconv.Itoa(filtered),
		"skipped":  strconv.Itoa(skipped),
	}
	return nil
}

type outcome int

const (
	outcomeSkipped outcome = iota
	outcomeFiltered
	outcomeImported
)

// importAlerts reads r one JSONL record per line, saving each as an Event on
// case cid. source names the Evidence the records came from, and is recorded
// on every Event as its Source. All saves happen in one transaction.
func importAlerts(store *model.Store, cid, source string, r io.Reader, severity string) (imported, filtered, skipped int, err error) {
	reader := bufio.NewReaderSize(r, 64*1024)
	txErr := store.Transaction(func(tx *model.Store) error {
		for {
			line, rerr := reader.ReadBytes('\n')
			if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
				out, ierr := importLine(tx, cid, source, trimmed, severity)
				if ierr != nil {
					return ierr
				}

				switch out {
				case outcomeSkipped:
					skipped++
				case outcomeFiltered:
					filtered++
				case outcomeImported:
					imported++
				}
			}

			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return rerr
			}
		}
		return nil
	})
	return imported, filtered, skipped, txErr
}

// importLine parses one alert record and, unless it's filtered by severity,
// saves it as an Event. A line that isn't valid JSON, or is missing message
// or datetime, is skipped rather than failing the whole import.
func importLine(tx *model.Store, cid, source string, line []byte, severity string) (outcome, error) {
	var rec map[string]any
	if err := json.Unmarshal(line, &rec); err != nil {
		slog.Warn("skipping malformed alert line", "module", "Alert Importer", "err", err)
		return outcomeSkipped, nil
	}

	message, _ := rec["message"].(string)
	datetime, _ := rec["datetime"].(string)
	if message == "" || datetime == "" {
		slog.Warn("skipping alert line missing message/datetime", "module", "Alert Importer")
		return outcomeSkipped, nil
	}

	t, err := parseDatetime(datetime)
	if err != nil {
		slog.Warn("skipping alert line with unparseable datetime", "module", "Alert Importer", "datetime", datetime, "err", err)
		return outcomeSkipped, nil
	}

	if severity != "all" {
		level, _ := lookupCI(rec, "level").(string)
		if !severityMatches(severity, level) {
			return outcomeFiltered, nil
		}
	}

	if desc, _ := rec["timestamp_desc"].(string); desc != "" {
		message = desc + " " + message
	}

	var raw bytes.Buffer
	if err := json.Indent(&raw, line, "", "  "); err != nil {
		return outcomeSkipped, nil
	}

	sum := sha1.Sum(append([]byte(source+"\x00"), line...))
	ev := model.Event{
		ID:         fmt.Sprintf("_alert_%x", sum),
		CaseID:     cid,
		Type:       "Other",
		Time:       model.Time(t),
		Event:      message,
		Raw:        raw.String(),
		Source:     source,
		Techniques: extractTechniques(rec),
	}

	if err := tx.SaveEvent(cid, ev, false); err != nil {
		return outcomeSkipped, err
	}
	return outcomeImported, nil
}

// parseDatetime accepts both a proper RFC3339 datetime (used by Chainsaw and
// Zircolite) and Hayabusa's space-separated variant.
func parseDatetime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05.999999999-07:00", s)
}

// severityMatches reports whether level clears the bar set by filter ("high"
// or "critical"); filter is assumed to already be one of AllowedSeverities.
func severityMatches(filter, level string) bool {
	level = strings.ToLower(level)
	switch filter {
	case "high":
		return level == "high" || level == "critical"
	case "critical":
		return level == "critical"
	default:
		return true
	}
}

// lookupCI returns rec[key], matching key case-insensitively - Hayabusa's raw
// "Level" field and Chainsaw/Zircolite's lowercase "level" both need to hit
// the same severity filter.
func lookupCI(rec map[string]any, key string) any {
	for k, v := range rec {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

// extractTechniques does a best-effort scan of rec's "tags" array (matched
// case-insensitively by key) for MITRE ATT&CK technique IDs, normalized to
// uppercase. It never errors - a record without a tags field, or without any
// recognizable technique tags, just yields no techniques.
func extractTechniques(rec map[string]any) model.Strings {
	arr, _ := lookupCI(rec, "tags").([]any)
	var techniques model.Strings
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			continue
		}
		if m := techniqueRe.FindStringSubmatch(s); m != nil {
			techniques = append(techniques, strings.ToUpper(m[1]))
		}
	}
	return techniques
}
