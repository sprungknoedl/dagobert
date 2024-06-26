package extensions

import (
	"archive/zip"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
)

var Extensions = []model.Extension{}

func Load() error {
	Extensions = append(Extensions, model.Extension{
		Name:        "Hayabusa",
		Description: "Hayabusa (隼) is a sigma-based threat hunting and fast forensics timeline generator for Windows event logs.",
		Supports:    func(e model.Evidence) bool { return slices.Contains([]string{".evtx", ".zip"}, filepath.Ext(e.Name)) },
		Run:         RunHayabusa,
	})
	return nil
}

func Get(name string) (model.Extension, error) {
	plugin, ok := fp.ToMap(Extensions, func(p model.Extension) string { return p.Name })[name]
	return plugin, fp.If(!ok, fmt.Errorf("invalid extension: %s", name), nil)
}

func addFromFS(store model.Store, obj model.Evidence) error {
	fr, err := os.Open(obj.Location)
	if err != nil {
		return err
	}

	stat, err := fr.Stat()
	if err != nil {
		return err
	}

	hasher := sha1.New()
	_, err = io.Copy(hasher, fr)
	if err != nil {
		return err
	}

	obj.Size = stat.Size()
	obj.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	return store.SaveEvidence(obj.CaseID, obj)
}

func clone(obj model.Evidence) (string, error) {
	sh, err := os.Open(obj.Location)
	if err != nil {
		return "", err
	}
	defer sh.Close()

	dh, err := os.CreateTemp("", "*."+obj.Name)
	if err != nil {
		return "", err
	}
	defer dh.Close()

	_, err = io.Copy(dh, sh)
	return dh.Name(), err
}

func unpack(obj model.Evidence) (string, error) {
	reader, err := zip.OpenReader(obj.Location)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}

	cleanup := func(err error) error {
		os.RemoveAll(dir) // try to cleanup but ignore if it fails
		return err
	}

	for _, file := range reader.File {
		dst := filepath.Clean(filepath.Join(dir, file.Name))

		// Check for file traversal attack
		if !strings.HasPrefix(dst, dir) {
			return "", cleanup(fmt.Errorf("invalid file path: %s", file.Name))
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, file.Mode()); err != nil {
				return "", cleanup(err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
				return "", cleanup(err)
			}

			destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return "", cleanup(err)
			}
			defer destFile.Close()

			srcFile, err := file.Open()
			if err != nil {
				return "", cleanup(err)
			}
			defer srcFile.Close()

			if _, err := io.Copy(destFile, srcFile); err != nil {
				return "", cleanup(err)
			}
		}
	}

	return dir, nil
}
