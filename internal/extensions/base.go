package extensions

import (
	"archive/zip"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/tty"
)

var Extensions = []model.Extension{}

func Load() error {
	Extensions = append(Extensions, model.Extension{
		Name:        "Hayabusa",
		Description: "Hayabusa (隼) is a sigma-based threat hunting and fast forensics timeline generator for Windows event logs.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".evtx" },
		Run:         RunHayabusaEvtx,
	})

	Extensions = append(Extensions, model.Extension{
		Name:        "Hayabusa",
		Description: "Hayabusa (隼) is a sigma-based threat hunting and fast forensics timeline generator for Windows event logs.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
		Run:         RunHayabusaZip,
	})

	Extensions = append(Extensions, model.Extension{
		Name:        "Plaso",
		Description: "Plaso (Plaso Langar Að Safna Öllu), or super timeline all the things, is a Python-based engine used by several tools for automatic creation of timelines.",
		Supports:    func(e model.Evidence) bool { return filepath.Ext(e.Name) == ".zip" },
		Run:         RunPlaso,
	})

	return nil
}

func Get(name string) (model.Extension, error) {
	plugin, ok := fp.ToMap(Extensions, func(p model.Extension) string { return p.Name })[name]
	return plugin, fp.If(!ok, fmt.Errorf("invalid extension: %s", name), nil)
}

func runDocker(src string, dst string, container string, args []string) error {
	srcmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), strings.TrimPrefix(filepath.Dir(src), "files/"))
	dstmnt := filepath.Join(os.Getenv("DOCKER_MOUNT"), strings.TrimPrefix(filepath.Dir(dst), "files/"))

	cmd := exec.Command("docker", append([]string{
		"run",
		"-v", srcmnt + ":/in:ro",
		"-v", dstmnt + ":/out",
		container,
	}, args...)...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("|%s| running command: docker %s", tty.Cyan(" DEB "), cmd.Args)
	return cmd.Run()
}

func addFromFS(store model.Store, obj model.Evidence) error {
	src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
	fr, err := os.Open(src)
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
	src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
	sh, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer sh.Close()

	dst := filepath.Join("files", "tmp", random(10)+"."+obj.Name)
	err = os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return "", err
	}

	dh, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer dh.Close()

	_, err = io.Copy(dh, sh)
	return dh.Name(), err
}

func unpack(obj model.Evidence) (string, error) {
	src := filepath.Join("files", "evidences", obj.CaseID, obj.Location)
	reader, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	dir := filepath.Join("files", "tmp", random(10))
	if err = os.MkdirAll(dir, 0755); err != nil {
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

	return dir + string(filepath.Separator), nil
}

func random(n int) string {
	// random string
	var src = rand.NewSource(time.Now().UnixNano())

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
