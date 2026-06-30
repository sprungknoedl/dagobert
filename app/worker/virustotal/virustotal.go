package virustotal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker/workerutils"
	vt "github.com/sprungknoedl/dagobert/pkg/virustotal"
)

type Module struct {
	client *vt.Client
}

func NewModule() *Module {
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
		slog.Info("module disabled, not configured", "module", "virustotal")
		return nil, errors.New("VIRUSTOTAL_APIKEY is not set, module disabled")
	}

	slog.Info("validating module prerequisites", "module", "virustotal")
	ctx, cancel := context.WithTimeout(context.Background(), workerutils.LookupTimeout)
	defer cancel()
	if err := m.client.Verify(ctx); err != nil {
		err = fmt.Errorf("connectivity check failed: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "virustotal", "err", err)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	ind, err := workerutils.GuardIndicatorRun(m, job)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, workerutils.LookupTimeout)
	defer cancel()

	res, err := m.client.Lookup(ctx, ind.Type, ind.Value)
	if err != nil {
		return err
	}

	// Always write at least the verdict (including "unknown") so the Success
	// state is meaningful.
	return store.SetEnrichment(model.Enrichment{
		CaseID:     job.Case.ID,
		ObjectType: "Indicator",
		ObjectID:   ind.ID,
		Module:     m.Name(),
		Verdict:    res.Verdict,
		Summary:    res.Summary,
		Link:       res.URL,
		FetchedAt:  model.Time(time.Now()),
	})
}

func (m *Module) RenderSettings() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return nil })
}
