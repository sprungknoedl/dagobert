package extensions

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
)

func runPlaso(store *model.Store, kase model.Case, obj model.Evidence, parsers string, ext string) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dst := filepath.Join("files", "evidences", obj.CaseID, name+ext)
	src, err := clone(obj)
	if err != nil {
		return err
	}
	defer os.Remove(src)

	err = runDocker(src, dst, "log2timeline/plaso", []string{
		"psteal.py",
		"--unattended",
		"--parsers", parsers,
		"--output-format", "dynamic",
		"--source", "/in/" + filepath.Base(src),
		"--storage-file", "/out/" + filepath.Base(dst),
		"--write", "/out/" + filepath.Base(dst) + ".csv",
	})

	if err != nil {
		// try to clean up
		os.Remove(dst)
		os.Remove(dst + ".csv")
		return err
	}

	if err := addFromFS("Plaso", store, kase, model.Evidence{
		ID:       random(10),
		CaseID:   obj.CaseID,
		Type:     "Other",
		Name:     filepath.Base(dst),
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: filepath.Base(dst),
	}); err != nil {
		return err
	}

	if err := addFromFS("Plaso", store, kase, model.Evidence{
		ID:       random(10),
		CaseID:   obj.CaseID,
		Type:     "Other",
		Name:     filepath.Base(dst) + ".csv",
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: filepath.Base(dst) + ".csv",
	}); err != nil {
		return err
	}

	return nil
}

func RunPlasoWindows(store *model.Store, kase model.Case, obj model.Evidence) error {
	return runPlaso(store, kase, obj, "win7", ".plaso")
}

func RunPlasoLinux(store *model.Store, kase model.Case, obj model.Evidence) error {
	return runPlaso(store, kase, obj, "linux", ".plaso")
}

func RunPlasoMacOS(store *model.Store, kase model.Case, obj model.Evidence) error {
	return runPlaso(store, kase, obj, "macos", ".plaso")
}

func RunPlasoMFT(store *model.Store, kase model.Case, obj model.Evidence) error {
	return runPlaso(store, kase, obj, "mft", ".mft.plaso")
}
