package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
)

var List = []Module{
	{
		Name:        "Hayabusa",
		Description: "Hayabusa (隼) is a sigma-based threat hunting and fast forensics timeline generator for Windows event logs.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".evtx" },
	},
	{
		Name:        "Plaso (Windows Preset)",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
	},
	{
		Name:        "Plaso (Linux Preset)",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
	},
	{
		Name:        "Plaso (MacOS Preset)",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
	},
	{
		Name:        "Plaso (Filesystem Timeline)",
		Description: "Run Plaso with the parser for NTFS $MFT metadata files to create a file system timeline that gives great insight into actions that occurred on the filesystem.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
	},
	{
		Name:        "Timesketch Importer",
		Description: "Timesketch is an open-source tool for collaborative forensic timeline analysis. Using sketches you and your collaborators can organize and work together.",
		Supports: func(e model.Evidence) bool {
			return strings.HasSuffix(e.Name, ".plaso") || strings.HasSuffix(e.Name, ".jsonl")
		},
	},
}

type Job struct {
	ID          string
	WorkerToken string
	Name        string
	Case        model.Case
	Evidence    model.Evidence

	Ctx context.Context
}

type Module struct {
	Name        string
	Description string
	Supports    func(model.Evidence) bool
}

func Get(name string) (Module, error) {
	plugin, ok := fp.ToMap(List, func(p Module) string { return p.Name })[name]
	return plugin, fp.If(!ok, fmt.Errorf("invalid extension: %s", name), nil)
}

func Supported(obj model.Evidence) []Module {
	return fp.Filter(List, func(p Module) bool { return p.Supports(obj) })
}

func Filepath(obj model.Evidence) string {
	return filepath.Join("files", "evidences", obj.CaseID, obj.Name)
}
