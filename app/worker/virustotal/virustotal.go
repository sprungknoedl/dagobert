package virustotal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	vt "github.com/sprungknoedl/dagobert/pkg/virustotal"
)

// lookupTimeout bounds a single VT lookup, derived from the job context so
// server shutdown cancels in-flight requests.
const lookupTimeout = 20 * time.Second

type Module struct {
	client *vt.Client
}

func NewModule() model.Module {
	return &Module{client: vt.NewClient(vt.Config{APIKey: os.Getenv("VIRUSTOTAL_APIKEY")})}
}

func (m *Module) Name() string {
	return "VirusTotal"
}

func (m *Module) Description() string {
	return "VirusTotal aggregates many antivirus products and online scan engines to inspect files, URLs, domains and IP addresses for known-malicious activity."
}

func (m *Module) Supports(obj any) bool {
	ind, ok := obj.(model.Indicator)
	if !ok {
		return false
	}
	switch ind.Type {
	case "IP", "Domain", "Hash", "URL": // VT covers all four
	default:
		return false
	}
	return ind.TLP != "TLP:RED" // never send not-shareable indicators out
}

func (m *Module) Validate() (model.Module, error) {
	if !m.client.Configured() {
		err := errors.New("VIRUSTOTAL_APIKEY is not set, module disabled")
		slog.Warn("validating module prerequisites failed", "module", "virustotal", "err", err)
		return nil, err
	}

	slog.Info("validating module prerequisites", "module", "virustotal")
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()
	if err := m.client.Verify(ctx); err != nil {
		err = fmt.Errorf("connectivity check failed: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "virustotal", "err", err)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(job model.Job) error {
	ind, ok := job.Object.Payload.(model.Indicator)
	if !ok {
		return fmt.Errorf("virustotal: unsupported type '%T'", job.Object.Payload)
	}

	// Run()-side egress gate: a TLP:RED value is never transmitted regardless
	// of how the job was scheduled.
	if !m.Supports(ind) {
		if ind.TLP == "TLP:RED" {
			return errors.New("indicator is TLP:RED — external enrichment disabled")
		}
		return errors.New("unsupported indicator type")
	}

	ctx, cancel := context.WithTimeout(job.Ctx, lookupTimeout)
	defer cancel()

	res, err := m.client.Lookup(ctx, ind.Type, ind.Value)
	if err != nil {
		return err
	}

	// Always write at least the verdict (including "unknown") so the Success
	// state is meaningful; SetIndicatorCustom skips the empty Link key.
	return job.Store.SetIndicatorCustom(job.Case.ID, ind.ID, map[string]string{
		"VirusTotal Enrichment": res.Summary,
		"VirusTotal Verdict":    res.Verdict,
		"VirusTotal Link":       res.URL,
	})
}

func (m *Module) CustomAttributes() []model.CustomAttribute {
	return []model.CustomAttribute{
		{Entity: "Indicator", Label: "VirusTotal Enrichment", Type: "textfield", Rank: 100},
		{Entity: "Indicator", Label: "VirusTotal Verdict", Type: "select", Options: model.Strings(model.EnrichmentVerdicts), Rank: 101},
		{Entity: "Indicator", Label: "VirusTotal Link", Type: "string", Rank: 102},
	}
}
