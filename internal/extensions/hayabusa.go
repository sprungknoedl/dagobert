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

func RunHayabusaEvtx(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))

	src, err := clone(obj)
	if err != nil {
		return err
	}
	defer os.Remove(src)
	log.Printf("|%s| hayabusa -> cloned file to %s", tty.Cyan(" DEB "), src)

	srcmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), strings.TrimPrefix(filepath.Dir(src), "files/"))
	dstmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), "evidences", obj.CaseID)

	cmd := exec.Command("docker", []string{
		"run",
		"-v", srcmnt + ":/in:ro",
		"-v", dstmnt + ":/out",
		"sprungknoedl/hayabusa",
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--file", "/in/" + filepath.Base(src),
		"--output", "/out/" + name + ".hayabusa.jsonl",
	}...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| hayabusa -> running command: docker %s", tty.Cyan(" DEB "), cmd.Args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(filepath.Join("files", "evidences", obj.CaseID, name+".hayabusa.jsonl"))
		return err
	}

	log.Printf("|%s| hayabusa -> successful run: %s", tty.Cyan(" DEB "), cmd.ProcessState)
	return addFromFS(store, model.Evidence{
		Type:     "Logs",
		Name:     name + ".hayabusa.jsonl",
		Source:   obj.Source,
		Notes:    "ext-hayabusa",
		Location: name + ".hayabusa.jsonl",
		CaseID:   obj.CaseID,
	})
}

func RunHayabusaZip(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))

	src, err := unpack(obj)
	if err != nil {
		return err
	}
	defer os.RemoveAll(src)
	log.Printf("|%s| hayabusa -> unpacked archive to %s", tty.Cyan(" DEB "), src)

	srcmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), strings.TrimPrefix(src, "files/"))
	dstmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), "evidences", obj.CaseID)

	cmd := exec.Command("docker", []string{
		"run",
		"-v", srcmnt + ":/in:ro",
		"-v", dstmnt + ":/out",
		"sprungknoedl/hayabusa",
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--directory", "/in/",
		"--output", "/out/" + name + ".hayabusa.jsonl",
	}...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| hayabusa -> running command: docker %s", tty.Cyan(" DEB "), cmd.Args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(filepath.Join("files", "evidences", obj.CaseID, name+".hayabusa.jsonl"))
		return err
	}

	log.Printf("|%s| hayabusa -> successful run: %s", tty.Cyan(" DEB "), cmd.ProcessState)
	return addFromFS(store, model.Evidence{
		Type:     "Logs",
		Name:     name + ".hayabusa.jsonl",
		Source:   obj.Source,
		Notes:    "ext-hayabusa",
		Location: name + ".hayabusa.jsonl",
		CaseID:   obj.CaseID,
	})
}
