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
	exe := os.Getenv("EXE_HAYABUSA")
	dst := filepath.Join("./files/evidences", filepath.Base(obj.CaseID),
		strings.TrimSuffix(obj.Name, filepath.Ext(obj.Name))+".hayabusa.json")
	log.Printf("|%s| hayabusa -> exe: %s", tty.Cyan(" DEB "), exe)
	log.Printf("|%s| hayabusa -> dst: %s", tty.Cyan(" DEB "), dst)

	// TODO: support archives
	args := []string{
		"json-timeline",
		"--JSONL-output",
		"--RFC-3339",
		"--UTC",
		"--no-wizard",
		"--min-level", "informational",
		"--profile", "timesketch-verbose",
		"--output", dst,
	}
	ext := filepath.Ext(obj.Name)
	switch ext {
	case ".zip":
		src, err := unpack(obj)
		if err != nil {
			return err
		}
		defer os.RemoveAll(src)
		log.Printf("|%s| hayabusa -> unpacked archive to %s", tty.Cyan(" DEB "), src)
		args = append(args, "--directory", src)
	case ".evtx":
		src, err := clone(obj)
		if err != nil {
			return err
		}
		defer os.Remove(src)
		log.Printf("|%s| hayabusa -> cloned file to %s", tty.Cyan(" DEB "), src)
		args = append(args, "--file", src)
	default:
		return fmt.Errorf("unsupported file type %s", ext)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| hayabusa -> running command", tty.Cyan(" DEB "))
	if err := cmd.Run(); err != nil {
		os.Remove(dst) // try to clean up
		return err
	}

	log.Printf("|%s| hayabusa -> successful run: %s", tty.Cyan(" DEB "), cmd.ProcessState)
	return addFromFS(store, model.Evidence{
		Type:     "Logs",
		Name:     filepath.Base(dst),
		Notes:    "ext-hayabusa",
		Location: dst,
		CaseID:   obj.CaseID,
	})
}
