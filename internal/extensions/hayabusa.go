package extensions

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
)

func RunHayabusaEvtx(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dst := filepath.Join("files", "evidences", obj.CaseID, name+".hayabusa.jsonl")
	src, err := clone(obj)
	if err != nil {
		return err
	}
	defer os.Remove(src)

	err = runDocker(src, dst, "sprungknoedl/hayabusa", []string{
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--file", "/in/" + filepath.Base(src),
		"--output", "/out/" + filepath.Base(dst),
	})

	if err != nil {
		// try to clean up
		os.Remove(dst)
		return err
	}

	return addFromFS(store, model.Evidence{
		Type:     "Logs",
		Name:     filepath.Base(dst),
		Source:   obj.Source,
		Notes:    "ext-hayabusa",
		Location: filepath.Base(dst),
		CaseID:   obj.CaseID,
	})
}

func RunHayabusaZip(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dst := filepath.Join("files", "evidences", obj.CaseID, name+".hayabusa.jsonl")

	src, err := unpack(obj)
	if err != nil {
		return err
	}
	defer os.RemoveAll(src)

	err = runDocker(src, dst, "sprungknoedl/hayabusa", []string{
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--directory", "/in/",
		"--output", "/out/" + filepath.Base(dst),
	})

	if err != nil {
		// try to clean up
		os.Remove(dst)
		return err
	}

	return addFromFS(store, model.Evidence{
		Type:     "Logs",
		Name:     filepath.Base(dst),
		Source:   obj.Source,
		Notes:    "ext-hayabusa",
		Location: filepath.Base(dst),
		CaseID:   obj.CaseID,
	})
}
