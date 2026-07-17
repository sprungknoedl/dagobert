package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// AttckRelease pins the MITRE ATT&CK STIX bundle version downloaded by
// `dagobert update`. This is the single source of truth for the pinned release.
const AttckRelease = "18.1"

const (
	mitreDir      = "mitre"
	mitreSentinel = mitreDir + "/.version"

	mitreBaseURL = "https://github.com/mitre-attack/attack-stix-data/raw/refs/heads/master"

	// The three ATT&CK matrices fetched by `dagobert update`, and their pinned
	// download URLs.
	enterpriseFile = "enterprise-attack.json"
	icsFile        = "ics-attack.json"
	mobileFile     = "mobile-attack.json"

	enterpriseURL = mitreBaseURL + "/enterprise-attack/enterprise-attack-" + AttckRelease + ".json"
	icsURL        = mitreBaseURL + "/ics-attack/ics-attack-" + AttckRelease + ".json"
	mobileURL     = mitreBaseURL + "/mobile-attack/mobile-attack-" + AttckRelease + ".json"
)

// updateMitre downloads the pinned MITRE ATT&CK data into mitre/ when it is
// missing or stale. A sentinel file (mitre/.version) records the version that
// was fetched: the download is skipped when every matrix is present and the
// sentinel already matches AttckRelease. force re-downloads unconditionally.
func updateMitre(force bool) error {
	if !force && mitreCurrent() {
		slog.Info("MITRE ATT&CK data current", "version", AttckRelease)
		return nil
	}

	if err := os.MkdirAll(mitreDir, 0755); err != nil {
		return err
	}

	for _, f := range []struct{ name, url string }{
		{enterpriseFile, enterpriseURL},
		{icsFile, icsURL},
		{mobileFile, mobileURL},
	} {
		dst := filepath.Join(mitreDir, f.name)
		slog.Info("Downloading MITRE ATT&CK data", "file", dst, "version", AttckRelease)
		if err := download(f.url, dst); err != nil {
			return fmt.Errorf("downloading %s: %w", f.name, err)
		}
	}

	return os.WriteFile(mitreSentinel, []byte(AttckRelease+"\n"), 0644)
}

// mitreCurrent reports whether every matrix is present and the sentinel matches
// the pinned release.
func mitreCurrent() bool {
	b, err := os.ReadFile(mitreSentinel)
	if err != nil || strings.TrimSpace(string(b)) != AttckRelease {
		return false
	}
	for _, name := range []string{enterpriseFile, icsFile, mobileFile} {
		if _, err := os.Stat(filepath.Join(mitreDir, name)); err != nil {
			return false
		}
	}
	return true
}

// download streams url to dst via a temp file + rename, so an interrupted
// download never leaves a truncated matrix in place.
func download(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if err := cmpErr(copyErr, closeErr); err != nil {
		if rerr := os.Remove(tmp); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			slog.Warn("failed to remove incomplete download", "err", rerr, "path", tmp)
		}
		return err
	}
	return os.Rename(tmp, dst)
}

func cmpErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
