package utils

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadZip fetches url's full body into memory so it can be opened as a
// zip.Reader, which needs random access (io.ReaderAt).
func DownloadZip(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// ExtractZipSubtree writes every entry of body under prefix to dst, discarding
// the rest of the archive. dst is removed first so a re-fetch is a clean
// overwrite rather than a merge with files from a previous version.
func ExtractZipSubtree(body []byte, prefix, dst string) error {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	if err := os.RemoveAll(dst); err != nil {
		return err
	}

	for _, f := range zr.File {
		rel, ok := strings.CutPrefix(f.Name, prefix)
		if !ok || rel == "" {
			continue
		}

		target := filepath.Join(dst, rel)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := copyZipEntry(f, target); err != nil {
			return err
		}
	}

	return nil
}

// ExtractZipFile writes the single entry name from body to dst, overwriting
// it. Unlike ExtractZipSubtree, it pulls out one file rather than a whole
// directory, for zip trees where the wanted file has unwanted siblings.
func ExtractZipFile(body []byte, name, dst string) error {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		return copyZipEntry(f, dst)
	}

	return fmt.Errorf("entry %q not found in archive", name)
}

func copyZipEntry(f *zip.File, target string) (err error) {
	fr, err := f.Open()
	if err != nil {
		return err
	}
	defer fr.Close()

	fw, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, fw.Close()) }()

	_, err = io.Copy(fw, fr)
	return err
}
