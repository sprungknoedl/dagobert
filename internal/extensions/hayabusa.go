package extensions

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

func RunHayabusa(store model.Store, obj model.Evidence) error {
	name := strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))
	dstdir := filepath.Dir(obj.Location)

	var srcdir string
	var args2 []string
	switch filepath.Ext(obj.Name) {
	case ".zip":
		src, err := unpack(obj)
		if err != nil {
			return err
		}
		defer os.RemoveAll(src)
		log.Printf("|%s| hayabusa -> unpacked archive to %s", tty.Cyan(" DEB "), src)
		srcdir = src
		args2 = []string{"--directory", "/in/"}

	case ".evtx":
		src, err := clone(obj)
		if err != nil {
			return err
		}
		defer os.Remove(src)
		log.Printf("|%s| hayabusa -> cloned file to %s", tty.Cyan(" DEB "), src)
		srcdir = filepath.Dir(src)
		args2 = []string{"--file", filepath.Join("/in/", filepath.Base(src))}

	default:
		return fmt.Errorf("unsupported file type %s", obj.Name)
	}

	volsrc, err := filepath.Abs(filepath.Join(os.Getenv("DAGOBERT_CWD"), srcdir))
	if err != nil {
		return err
	}

	voldst, err := filepath.Abs(filepath.Join(os.Getenv("DAGOBERT_CWD"), dstdir))
	if err != nil {
		return err
	}

	args := append([]string{
		"run",
		"-v", volsrc + ":/in:ro",
		"-v", voldst + ":/out",
		"sprungknoedl/hayabusa",
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--output", filepath.Join("/out/", name+".hayabusa.jsonl"),
	}, args2...)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| hayabusa -> running command: docker %s", tty.Cyan(" DEB "), args)
	if err := cmd.Run(); err != nil {
		// try to clean up
		os.Remove(filepath.Join(dstdir, name+".hayabusa.jsonl"))
		return err
	}

	log.Printf("|%s| hayabusa -> successful run: %s", tty.Cyan(" DEB "), cmd.ProcessState)
	return addFromFS(store, model.Evidence{
		Type:     "Logs",
		Name:     name + ".hayabusa.jsonl",
		Source:   obj.Source,
		Notes:    "ext-hayabusa",
		Location: filepath.Join(dstdir, name+".hayabusa.jsonl"),
		CaseID:   obj.CaseID,
	})
}
