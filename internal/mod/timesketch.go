package mod

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

func init() {
	if os.Getenv("TIMESKETCH_URL") != "" {
		ts, err := timesketch.NewClient(
			os.Getenv("TIMESKETCH_URL"),
			os.Getenv("TIMESKETCH_USER"),
			os.Getenv("TIMESKETCH_PASS"),
		)
		if err != nil {
			log.Printf("Failed to create timesketch client: %v", err)
			return
		}

		Register(model.Mod{
			Name:        "Upload Timeline to Timesketch",
			Description: "Timesketch is an open-source tool for collaborative forensic timeline analysis. Using sketches you and your collaborators can organize and work together.",
			Supports: func(e model.Evidence) bool {
				return strings.HasSuffix(e.Name, ".plaso") || strings.HasSuffix(e.Name, ".jsonl")
			},
			Run: UploadToTimesketch(ts),
		})
	}
}

func UploadToTimesketch(ts *timesketch.Client) func(store *model.Store, kase model.Case, obj model.Evidence) error {
	return func(store *model.Store, kase model.Case, obj model.Evidence) error {
		if ts == nil || kase.SketchID == 0 {
			return errors.New("invalid timesketch configuration")
		}

		src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
		return ts.Upload(kase.SketchID, src)
	}
}
