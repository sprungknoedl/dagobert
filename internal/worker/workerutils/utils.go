package workerutils

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

// LookupTimeout bounds a single lookup, derived from the job context so
// server shutdown cancels in-flight requests.
const LookupTimeout = 20 * time.Second

// OnEvidenceAdded is wired to the hooks engine by the worker package; this
// package can not import worker directly, because the module packages that
// import workerutils are themselves imported by worker.
var OnEvidenceAdded func(store *model.Store, obj model.Evidence)

func Filepath(obj model.Evidence) string {
	return filepath.Join("files", "evidences", obj.CaseID, obj.Name)
}

func AddFromFS(store *model.Store, obj model.Evidence) error {
	fr, err := os.Open(Filepath(obj))
	if err != nil {
		return err
	}
	defer fr.Close()

	stat, err := fr.Stat()
	if err != nil {
		return err
	}

	hasher := sha1.New()
	if _, err := io.Copy(hasher, fr); err != nil {
		return err
	}

	obj.ID = fp.Random(10)
	obj.Size = stat.Size()
	obj.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	if err := store.SaveEvidence(obj.CaseID, obj); err != nil {
		return err
	}

	if OnEvidenceAdded != nil {
		OnEvidenceAdded(store, obj)
		return nil
	} else {
		return fmt.Errorf("OnEvidenceAdded not wired in; failure in package setup")
	}
}

func GuardEvidenceRun(m model.Module, job model.Job) (model.Evidence, error) {
	obj, ok := job.Object.Payload.(model.Evidence)
	if !ok {
		return model.Evidence{}, fmt.Errorf("%s: unsupported type '%T'", m.Name(), job.Object.Payload)
	}

	if !m.Supports(obj) {
		return model.Evidence{}, errors.New("unsupported indicator type")
	}

	return obj, nil
}

func GuardIndicatorRun(m model.Module, job model.Job) (model.Indicator, error) {
	obj, ok := job.Object.Payload.(model.Indicator)
	if !ok {
		return model.Indicator{}, fmt.Errorf("%s: unsupported type '%T'", m.Name(), job.Object.Payload)
	}

	if !m.Supports(obj) {
		if obj.TLP == "TLP:RED" {
			return model.Indicator{}, errors.New("indicator is TLP:RED — external enrichment disabled")
		}
		return model.Indicator{}, errors.New("unsupported indicator type")
	}

	return obj, nil
}
