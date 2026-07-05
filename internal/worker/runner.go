package worker

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker/abuseipdb"
	"github.com/sprungknoedl/dagobert/internal/worker/hayabusa"
	"github.com/sprungknoedl/dagobert/internal/worker/hybridanalysis"
	"github.com/sprungknoedl/dagobert/internal/worker/plaso"
	"github.com/sprungknoedl/dagobert/internal/worker/timesketch"
	"github.com/sprungknoedl/dagobert/internal/worker/virustotal"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	tsclient "github.com/sprungknoedl/dagobert/pkg/timesketch"
	"gorm.io/gorm"
)

var Modules = map[string]model.Module{}

func Supported(obj any) []model.Module {
	return fp.ToList(fp.FilterM(Modules, func(p model.Module) bool { return p.Supports(obj) }))
}

// Start validates modules and launches the runner pool. Called from
// handler.Run; ctx is the server's shutdown context, ts the shared
// Timesketch client.
func Start(ctx context.Context, store *model.Store, ts *tsclient.Client) {
	for _, m := range []model.Module{
		abuseipdb.NewModule(),
		hayabusa.NewModule(),
		hybridanalysis.NewModule(),
		plaso.NewModule(),
		timesketch.NewModule(ts),
		virustotal.NewModule(),
	} {
		Modules[m.Name()] = m
	}

	slog.Debug("Loading modules")
	modules := map[string]model.Module{}
	for _, name := range slices.Sorted(maps.Keys(Modules)) {
		if _, err := Modules[name].Validate(); err == nil {
			modules[name] = Modules[name]
		}
	}

	if len(modules) == 0 {
		slog.Warn("no job modules available — configure MODULE_* env vars")
		return
	}

	num, err := strconv.Atoi(cmp.Or(os.Getenv("DAGOBERT_WORKERS"), "3"))
	if err != nil || num < 0 || num > 1000 {
		slog.Warn("Invalid number of workers, falling back to default of 3", "num", num, "err", err)
		num = 3
	}

	slog.Info("Starting job runners", "num", num, "modules", fp.Keys(modules))
	for range num {
		go runner(ctx, store, modules)
	}

	slog.Debug("Loading hooks")
	err = LoadHooks(store)
	if err != nil {
		slog.Error("Failed to load hooks", "err", err)
		return
	}

	slog.Debug("Rescheduling stale jobs")
	err = store.RescheduleStaleJobs()
	if err != nil {
		slog.Error("Failed to reschedule state jobs", "err", err)
	}
}

func runner(ctx context.Context, store *model.Store, modules map[string]model.Module) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			job, err := store.PopJob(fp.Keys(modules))
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			} else if err != nil {
				slog.Error("failed to fetch job", "err", err)
				continue
			}

			slog.Info("running job", "job", job.ID, "module", job.Name)
			// PopJob returns the job's own columns but not the Case
			// association, so load it here. After this, modules can rely on
			// job.Case being populated.
			errmsg := ""
			if kase, err := store.GetCase(job.CaseID); err != nil {
				errmsg = err.Error()
				slog.Warn("failed to load job case", "job", job.ID, "case", job.CaseID, "err", err)
			} else {
				job.Case = kase
				if err := modules[job.Name].Run(ctx, store, job); err != nil {
					errmsg = err.Error()
					slog.Warn("failed to process job", "job", job.ID, "module", job.Name, "err", err)
				}
			}

			err = store.AckJob(model.Job{
				ID:     job.ID,
				Status: fp.If(errmsg != "", "Failed", "Success"),
				Error:  errmsg,
			})
			if err != nil {
				slog.Warn("failed to ack job", "job", job.ID, "module", job.Name, "err", err)
			}
		}
	}
}
