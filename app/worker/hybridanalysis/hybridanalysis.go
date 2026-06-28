package hybridanalysis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	ha "github.com/sprungknoedl/dagobert/pkg/hybridanalysis"
)

const lookupTimeout = 20 * time.Second

type Module struct {
	client *ha.Client
}

func NewModule() model.Module {
	return &Module{client: ha.NewClient(ha.Config{APIKey: os.Getenv("HYBRIDANALYSIS_APIKEY")})}
}

func (m *Module) Name() string { return "Hybrid Analysis" }

func (m *Module) Description() string {
	return "Hybrid Analysis (Falcon Sandbox) is a free malware analysis service that detects and analyzes unknown threats using a unique Hybrid Analysis technology."
}

func (m *Module) Supports(obj any) bool {
	ind, ok := obj.(model.Indicator)
	if !ok {
		return false
	}
	if ind.Type != "Hash" {
		return false
	}
	return ind.TLP != "TLP:RED"
}

func (m *Module) Validate() (model.Module, error) {
	if !m.client.Configured() {
		err := errors.New("HYBRIDANALYSIS_APIKEY is not set, module disabled")
		slog.Warn("validating module prerequisites failed", "module", "hybridanalysis", "err", err)
		return nil, err
	}

	slog.Info("validating module prerequisites", "module", "hybridanalysis")
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()
	if err := m.client.Verify(ctx); err != nil {
		err = fmt.Errorf("connectivity check failed: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "hybridanalysis", "err", err)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	ind, ok := job.Object.Payload.(model.Indicator)
	if !ok {
		return fmt.Errorf("hybridanalysis: unsupported type '%T'", job.Object.Payload)
	}

	if !m.Supports(ind) {
		if ind.TLP == "TLP:RED" {
			return errors.New("indicator is TLP:RED — external enrichment disabled")
		}
		return errors.New("unsupported indicator type")
	}

	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
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
