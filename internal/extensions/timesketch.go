package extensions

import (
	"errors"
	"path/filepath"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

func UploadToTimesketch(ts *timesketch.Client) func(store *model.Store, kase model.Case, obj model.Evidence) error {
	return func(store *model.Store, kase model.Case, obj model.Evidence) error {
		if ts == nil || kase.SketchID == 0 {
			return errors.New("invalid timesketch configuration")
		}

		src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
		return ts.Upload(kase.SketchID, src)
	}
}
