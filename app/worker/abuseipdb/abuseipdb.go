package abuseipdb

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
	ab "github.com/sprungknoedl/dagobert/pkg/abuseipdb"
)

type Module struct {
	client *ab.Client
}

func NewModule() *Module {
	return &Module{client: ab.NewClient(ab.Config{APIKey: os.Getenv("ABUSEIPDB_APIKEY")})}
}

func (m *Module) Name() string { return "AbuseIPDB" }

func (m *Module) Description() string {
	return "AbuseIPDB is an IP reputation database that aggregates user-reported abuse to identify malicious IP addresses."
}

func (m *Module) Supports(obj any) bool {
	ind, ok := obj.(model.Indicator)
	if !ok {
		return false
	}
	if ind.Type != "IP" {
		return false
	}
	return ind.TLP != "TLP:RED"
}

func (m *Module) Validate() (model.Module, error) {
	if !m.client.Configured() {
		slog.Info("module disabled, not configured", "module", "abuseipdb")
		return nil, errors.New("ABUSEIPDB_APIKEY is not set, module disabled")
	}

	slog.Info("validating module prerequisites", "module", "abuseipdb")
	ctx, cancel := context.WithTimeout(context.Background(), workerutils.LookupTimeout)
	defer cancel()
	if err := m.client.Verify(ctx); err != nil {
		err = fmt.Errorf("connectivity check failed: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "abuseipdb", "err", err)
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

	res, err := m.client.Lookup(ctx, ind.Value)
	if err != nil {
		return err
	}

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
