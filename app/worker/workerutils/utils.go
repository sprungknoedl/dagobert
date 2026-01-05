package workerutils

import (
	"archive/zip"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func Filepath(obj model.Evidence) string {
	return filepath.Join("files", "evidences", obj.CaseID, obj.Name)
}

func AddFromFS(obj model.Evidence) error {
	body := bytes.NewBuffer(nil)
	form := multipart.NewWriter(body)

	err := errors.Join(
		form.WriteField("Name", obj.Name),
		form.WriteField("Type", obj.Type),
		form.WriteField("Source", obj.Source),
		form.WriteField("Notes", obj.Notes))
	if err != nil {
		return err
	}

	err = form.Close()
	if err != nil {
		return err
	}

	uri := os.Getenv("DAGOBERT_URL") + "/cases/" + obj.CaseID + "/evidences/" + cmp.Or(obj.ID, "new")
	req, err := http.NewRequest(http.MethodPost, uri, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("X-API-Key", os.Getenv("DAGOBERT_API_KEY"))
	client := http.Client{}
	_, err = client.Do(req)
	return err
}

func unpack(obj model.Evidence) (string, error) {
	src := filepath.Join("files", "evidences", obj.CaseID, obj.Name)
	reader, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	dir := filepath.Join("files", "tmp", fp.Random(10))
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
