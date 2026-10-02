package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/zip"
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

// maxArchiveContentSize caps the total *decompressed* size of the .evtx files
// written while extracting an archive, mirroring
// internal/handler/archive.go's function of the same name (duplicated here
// because internal/modules can't import internal/handler). Override with
// MAX_ARCHIVE_CONTENT_SIZE (bytes).
func maxArchiveContentSize() int64 {
	if v := os.Getenv("MAX_ARCHIVE_CONTENT_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 10 << 30 // 10 GiB
}

// extractEvtx writes every .evtx entry of zr under dstDir, preserving each
// entry's relative path, and returns the number of files written. Every
// other entry is skipped. An entry whose cleaned relative path would escape
// dstDir is rejected (zip-slip). password is applied to encrypted entries; an
// encrypted entry with no password set fails with a clear error. The total
// decompressed bytes written across all entries is capped at budget.
func extractEvtx(zr *zip.Reader, dstDir, password string, budget int64) (count int, err error) {
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || strings.ToLower(filepath.Ext(f.Name)) != ".evtx" {
			continue
		}
		if !filepath.IsLocal(f.Name) {
			return 0, fmt.Errorf("illegal entry path in archive: %q", f.Name)
		}

		if f.IsEncrypted() {
			if password == "" {
				return 0, fmt.Errorf("entry %q is password protected", f.Name)
			}
			f.SetPassword(password)
		}

		target := filepath.Join(dstDir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return 0, err
		}

		n, err := extractEvtxEntry(f, target, budget)
		if err != nil {
			if errors.Is(err, zip.ErrPassword) {
				return 0, fmt.Errorf("entry %q: wrong password", f.Name)
			}
			return 0, fmt.Errorf("entry %q: %w", f.Name, err)
		}
		budget -= n
		count++
	}

	return count, nil
}

func extractEvtxEntry(f *zip.File, target string, budget int64) (n int64, err error) {
	fr, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer fr.Close()

	fw, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, fw.Close()) }()

	// read one byte past the budget so a copy that stops exactly at budget is
	// distinguishable from one that would have overrun
	n, err = io.Copy(fw, io.LimitReader(fr, budget+1))
	if err != nil {
		return n, err
	}
	if n > budget {
		return n, fmt.Errorf("archive exceeds maximum decompressed size of %d bytes", maxArchiveContentSize())
	}
	return n, nil
}

// EvtxInput resolves evidence to a filesystem path an EVTX-scanning module can
// run against, plus a cleanup function the caller must defer. For .evtx
// evidence it returns Filepath(evidence) unchanged with a no-op cleanup. For
// .zip evidence it extracts every .evtx entry into a fresh temp directory
// under model.TmpDir and returns that directory; cleanup removes it. On its
// own error paths (empty archive, extraction failure) it removes the temp
// directory itself before returning the error.
func EvtxInput(evidence model.Evidence) (string, func(), error) {
	noop := func() {}

	src := Filepath(evidence)
	if filepath.Ext(evidence.Name) != ".zip" {
		return src, noop, nil
	}

	zr, err := zip.OpenReader(src)
	if err != nil {
		return "", noop, err
	}
	defer zr.Close()

	if err := os.MkdirAll(model.TmpDir, 0755); err != nil {
		return "", noop, err
	}
	dir, err := os.MkdirTemp(model.TmpDir, "evtx-extract-*")
	if err != nil {
		return "", noop, err
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.Warn("failed to remove evtx extraction temp dir", "err", err, "path", dir)
		}
	}

	count, err := extractEvtx(&zr.Reader, dir, evidence.Password, maxArchiveContentSize())
	if err != nil {
		cleanup()
		return "", noop, err
	}
	if count == 0 {
		cleanup()
		return "", noop, errors.New("no .evtx files found in archive")
	}

	return dir, cleanup, nil
}
