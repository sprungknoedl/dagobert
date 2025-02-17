package mod

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
)

func init() {
	Register(model.Mod{
		Name:        "Plaso (Windows Preset)",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
		Run:         RunPlasoWindows,
	})

	Register(model.Mod{
		Name:        "Plaso (Linux Preset)",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
		Run:         RunPlasoLinux,
	})

	Register(model.Mod{
		Name:        "Plaso (MacOS Preset)",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
		Run:         RunPlasoMacOS,
	})

	Register(model.Mod{
		Name:        "Plaso (Filesystem Timeline)",
		Description: "Run Plaso with the parser for NTFS $MFT metadata files to create a file system timeline that gives great insight into actions that occurred on the filesystem.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
		Run:         RunPlasoMFT,
	})

}

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
