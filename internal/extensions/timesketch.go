package extensions

import (
	"os"
	"path/filepath"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

func UploadToTimesketch() (func(store *model.Store, kase model.Case, obj model.Evidence) error, error) {
	client, err := timesketch.NewClient(
		os.Getenv("TIMESKETCH_URL"),
		os.Getenv("TIMESKETCH_USER"),
		os.Getenv("TIMESKETCH_PASS"),
	)
	if err != nil {
		return nil, err
	}

	return func(store *model.Store, kase model.Case, obj model.Evidence) error {
		src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
		return client.Upload(1, src)
	}, nil
}
