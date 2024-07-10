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

	src, err := clone(obj)
	if err != nil {
		return err
	}
	defer os.Remove(src)

	srcmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), strings.TrimPrefix(filepath.Dir(src), "files/"))
	dstmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), "evidences", obj.CaseID)

	// psteal.py --source /path/to/artifact -o dynamic --storage-file $artifact_id.plaso -w $artifact_id.csv
	args := []string{
		"run",
		"-v", srcmnt + ":/in:ro",
		"-v", dstmnt + ":/out",
		"log2timeline/plaso",
		"psteal.py",
		"--unattended",
		"--parsers", "prefetch",
		"--output-format", "dynamic",
		"--source", "/in/" + filepath.Base(src),
		"--storage-file", "/out/" + name + ".plaso",
		"--write", "/out/" + name + ".plaso.csv",
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| plaso -> running command: %s", tty.Cyan(" DEB "), cmd)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(filepath.Join("files", "evidences", obj.CaseID, name+".plaso"))
		os.Remove(filepath.Join("files", "evidences", obj.CaseID, name+".plaso.csv"))
		return err
	}

	log.Printf("|%s| plaso -> successful run: %s", tty.Cyan(" DEB "), cmd.ProcessState)
	if err := addFromFS(store, model.Evidence{
		Type:     "Other",
		Name:     name + ".plaso",
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: name + ".plaso",
		CaseID:   obj.CaseID,
	}); err != nil {
		return err
	}

	if err := addFromFS(store, model.Evidence{
		Type:     "Other",
		Name:     name + ".plaso.csv",
		Source:   obj.Source,
		Notes:    "ext-plaso",
		Location: name + ".plaso.csv",
		CaseID:   obj.CaseID,
	}); err != nil {
		return err
	}

	return nil
}
