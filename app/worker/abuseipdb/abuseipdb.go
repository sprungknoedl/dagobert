package abuseipdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	ab "github.com/sprungknoedl/dagobert/pkg/abuseipdb"
)

const lookupTimeout = 20 * time.Second

type Module struct {
	client *ab.Client
}

func NewModule() model.Module {
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
		err := errors.New("ABUSEIPDB_APIKEY is not set, module disabled")
		slog.Warn("validating module prerequisites failed", "module", "abuseipdb", "err", err)
		return nil, err
	}

	slog.Info("validating module prerequisites", "module", "abuseipdb")
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()
	if err := m.client.Verify(ctx); err != nil {
		err = fmt.Errorf("connectivity check failed: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "abuseipdb", "err", err)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(job model.Job) error {
	ind, ok := job.Object.Payload.(model.Indicator)
	if !ok {
		return fmt.Errorf("abuseipdb: unsupported type '%T'", job.Object.Payload)
	}

	if !m.Supports(ind) {
		if ind.TLP == "TLP:RED" {
			return errors.New("indicator is TLP:RED — external enrichment disabled")
		}
		return errors.New("unsupported indicator type")
	}

	ctx, cancel := context.WithTimeout(job.Ctx, lookupTimeout)
	defer cancel()

	res, err := m.client.Lookup(ctx, ind.Value)
	if err != nil {
		return err
	}

	return job.Store.SetEnrichment(model.Enrichment{
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
