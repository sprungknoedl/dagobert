package extensions

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

func RunPlaso(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dstdir := filepath.Dir(obj.Location)

	src, err := clone(obj)
	if err != nil {
		return err
	}
	defer os.Remove(src)

	volsrc, err := filepath.Abs(filepath.Join(os.Getenv("DAGOBERT_CWD"), filepath.Dir(src)))
	if err != nil {
		return err
	}

	voldst, err := filepath.Abs(filepath.Join(os.Getenv("DAGOBERT_CWD"), dstdir))
	if err != nil {
		return err
	}

	log.Printf("|%s| plaso -> cloned file to %s", tty.Cyan(" DEB "), src)
	// psteal.py --source /path/to/artifact -o dynamic --storage-file $artifact_id.plaso -w $artifact_id.csv
	args := []string{
		"run",
		"-v", volsrc + ":/in:ro",
		"-v", voldst + ":/out",
		"log2timeline/plaso",
		"psteal.py",
		"--unattended",
		"--parsers", "prefetch",
		"--source", filepath.Join("/in", filepath.Base(src)),
		"--output-format", "dynamic",
		"--storage-file", filepath.Join("/out", name+".plaso"),
		"--write", filepath.Join("/out", name+".plaso.csv"),
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| plaso -> running command: %s", tty.Cyan(" DEB "), cmd)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(filepath.Join(dstdir, name+".plaso"))
		os.Remove(filepath.Join(dstdir, name+".plaso.csv"))
		return err
	}

	log.Printf("|%s| plaso -> successful run: %s", tty.Cyan(" DEB "), cmd.ProcessState)
	if err := addFromFS(store, model.Evidence{
		Type:     "Other",
		Name:     name + ".plaso",
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: filepath.Join(dstdir, name+".plaso"),
		CaseID:   obj.CaseID,
	}); err != nil {
		return err
	}

	if err := addFromFS(store, model.Evidence{
		Type:     "Other",
		Name:     name + ".plaso.csv",
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: filepath.Join(dstdir, name+".plaso.csv"),
		CaseID:   obj.CaseID,
	}); err != nil {
		return err
	}

	return nil
}
