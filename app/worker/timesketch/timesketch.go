package timesketch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker/workerutils"
	ts "github.com/sprungknoedl/dagobert/pkg/timesketch"
)

type Module struct {
	client *ts.Client
}

func NewModule(client *ts.Client) model.Module {
	return &Module{client: client}
}

func (m *Module) Name() string {
	return "Timesketch Importer"
}

func (m *Module) Description() string {
	return "Timesketch is an open-source tool for collaborative forensic timeline analysis. Using sketches you and your collaborators can organize and work together."
}

func (m *Module) Supports(obj any) bool {
	if e, ok := obj.(model.Evidence); ok {
		return strings.HasSuffix(e.Name, ".plaso") || strings.HasSuffix(e.Name, ".jsonl")
	}
	return false
}

func (m *Module) Validate() (model.Module, error) {
	if !m.client.Configured() {
		err := errors.New("TIMESKETCH_URL is not set, module disabled")
		slog.Warn("validating module prerequisites failed", "module", "timesketch", "err", err)
		return nil, err
	}

	slog.Info("validating module prerequisites", "module", "timesketch")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.client.Login(ctx); err != nil {
		err = fmt.Errorf("login failed: %w", err)
		slog.Warn("validating module prerequisites failed", "module", "timesketch", "err", err)
		return nil, err
	}

	return m, nil
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	evidence, ok := job.Object.Payload.(model.Evidence)
	if !ok {
		return fmt.Errorf("timesketch: unsupported type '%T'", job.Object.Payload)
	}
	if job.Case.SketchID == 0 {
		return errors.New("timesketch: case is not linked to a sketch")
	}

	src := workerutils.Filepath(evidence)
	return m.client.Upload(ctx, job.Case.SketchID, src)
}
