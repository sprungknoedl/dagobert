package handler

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

// slugify turns an incident title into a filesystem-safe filename stem,
// collapsing runs of non-alphanumeric characters into single hyphens.
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// maxArchiveSize caps the size of an uploaded case archive to bound memory and
// disk use on import. Override with MAX_ARCHIVE_SIZE (bytes).
func maxArchiveSize() int64 {
	if v := os.Getenv("MAX_ARCHIVE_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 10 << 30 // 10 GiB
}

// maxArchiveContentSize caps the total *decompressed* size of the binaries
// written during an import. maxArchiveSize bounds only the compressed upload, so
// without this a zip bomb (100x+ expansion) could exhaust the disk. A real case
// with dozens of evidence files and malware samples is legitimately large, so
// the default is generous. Override with MAX_ARCHIVE_CONTENT_SIZE (bytes).
func maxArchiveContentSize() int64 {
	if v := os.Getenv("MAX_ARCHIVE_CONTENT_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 10 << 30 // 10 GiB
}

// ExportArchive streams a single case to a ZIP archive. With binaries=true the
// evidence files and malware samples are bundled too; the default (false) is a
// metadata-only archive for handoff/archival.
func (h *Handler) ExportArchive(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	includeBinaries := r.URL.Query().Get("binaries") == "true"

	kase, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	user := GetUser(r)
	exportedBy := fp.If(user.Email != "", user.Email, fp.If(user.UPN != "", user.UPN, user.Name))

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	sourceInstance := scheme + "://" + r.Host

	filename := fmt.Sprintf("%s-%s-%s.zip",
		fp.If(slugify(kase.Name) != "", slugify(kase.Name), cid),
		time.Now().Format("20060102"),
		fp.If(includeBinaries, "full", "metadata"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	// stream directly to the client; evidence binaries can be large
	if err := writeCaseArchive(w, h.Store, cid, sourceInstance, exportedBy, includeBinaries); err != nil {
		// headers/body are already (partially) committed, so we can only log
		slog.Error("failed to export case archive", "err", err, "case", cid)
	}
}

// writeCaseArchive serializes the case graph and (optionally) its binaries into
// a ZIP written to out.
func writeCaseArchive(out io.Writer, store *model.Store, cid, sourceInstance, exportedBy string, includeBinaries bool) error {
	arch, err := store.ExportCaseArchive(cid)
	if err != nil {
		return err
	}

	schemaVer, err := model.SchemaVersion()
	if err != nil {
		return err
	}

	// resolve the binary entries up-front so the manifest (written first) can
	// carry per-file warnings for anything missing on disk
	type binEntry struct{ name, disk string }
	var bins []binEntry
	var warnings []string
	if includeBinaries {
		for _, e := range arch.Evidences {
			disk := filepath.Join("files", "evidences", cid, e.Name)
			if _, err := os.Stat(disk); err != nil {
				warnings = append(warnings, fmt.Sprintf("evidence %q: binary missing on disk", e.Name))
				continue
			}
			bins = append(bins, binEntry{"evidences/" + e.Name, disk})
		}
		for _, m := range arch.Malware {
			disk := filepath.Join("files", "malware", cid, m.Hash+".zip")
			if _, err := os.Stat(disk); err != nil {
				warnings = append(warnings, fmt.Sprintf("malware %q: sample missing on disk", m.Hash))
				continue
			}
			bins = append(bins, binEntry{"malware/" + m.Hash + ".zip", disk})
		}
	}

	manifest := model.Manifest{
		Format:           model.ArchiveFormat,
		FormatVersion:    model.ArchiveFormatVersion,
		SchemaVersion:    schemaVer,
		ExportedAt:       model.Time(time.Now().UTC()),
		ExportedBy:       exportedBy,
		SourceInstance:   sourceInstance,
		IncludesBinaries: includeBinaries,
		OriginalCaseID:   arch.Case.ID,
		Counts: map[string]int{
			"assets":     len(arch.Assets),
			"events":     len(arch.Events),
			"evidences":  len(arch.Evidences),
			"indicators": len(arch.Indicators),
			"malware":    len(arch.Malware),
			"notes":      len(arch.Notes),
			"tasks":      len(arch.Tasks),
		},
		Warnings: warnings,
	}

	zw := zip.NewWriter(out)
	if err := writeJSONEntry(zw, "manifest.json", manifest); err != nil {
		return err
	}
	if err := writeJSONEntry(zw, "case.json", arch); err != nil {
		return err
	}
	for _, b := range bins {
		if err := writeFileEntry(zw, b.name, b.disk); err != nil {
			return err
		}
	}
	return zw.Close()
}

func writeJSONEntry(zw *zip.Writer, name string, v any) error {
	fw, err := zw.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fw)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeFileEntry(zw *zip.Writer, name, disk string) error {
	fw, err := zw.Create(name)
	if err != nil {
		return err
	}
	fr, err := os.Open(disk)
	if err != nil {
		return err
	}
	defer fr.Close()
	_, err = io.Copy(fw, fr)
	return err
}

// ImportArchiveForm shows the upload form for a case archive.
func (h *Handler) ImportArchiveForm(w http.ResponseWriter, r *http.Request) {
	Render(w, r, http.StatusOK, views.ImportArchiveDialog())
}

// ImportArchive handles both stages of the import: the initial upload (parse the
// manifest, stage the file, show a confirmation screen) and the final commit
// (recreate the case, restore binaries) once the operator confirms.
func (h *Handler) ImportArchive(w http.ResponseWriter, r *http.Request) {
	if token := r.FormValue("token"); r.FormValue("confirm") == "yes" && token != "" {
		h.CommitImport(w, r, token)
		return
	}
	h.PreviewImport(w, r)
}

// PreviewImport reads the uploaded archive, validates it, stages it to disk and
// renders the confirmation screen.
func (h *Handler) PreviewImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxArchiveSize())
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		Warn(w, r, fmt.Errorf("archive too large or malformed upload: %w", err))
		return
	}

	fr, _, err := r.FormFile("file")
	if err != nil {
		Warn(w, r, err)
		return
	}
	defer fr.Close()

	// stream the upload to a staging file so we get random access (zip needs a
	// ReaderAt) without buffering the whole archive in memory
	token := fp.Random(24)
	staged := stagedArchivePath(token)
	if err := os.MkdirAll(filepath.Dir(staged), 0755); err != nil {
		Err(w, r, err)
		return
	}
	fw, err := os.OpenFile(staged, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		Err(w, r, err)
		return
	}
	if _, err := io.Copy(fw, fr); err != nil {
		fw.Close()
		os.Remove(staged)
		Warn(w, r, err)
		return
	}
	fw.Close()

	zr, err := zip.OpenReader(staged)
	if err != nil {
		os.Remove(staged)
		Warn(w, r, fmt.Errorf("not a valid zip archive: %w", err))
		return
	}
	defer zr.Close()

	manifest, err := readManifest(&zr.Reader)
	if err != nil {
		os.Remove(staged)
		Warn(w, r, err)
		return
	}
	if err := validateManifest(manifest); err != nil {
		os.Remove(staged)
		Warn(w, r, err)
		return
	}
	// decode the graph and check entry paths now so the confirmation screen is
	// only shown for an archive we can actually import
	if _, err := readCaseArchive(&zr.Reader); err != nil {
		os.Remove(staged)
		Warn(w, r, err)
		return
	}
	if err := validateArchivePaths(&zr.Reader); err != nil {
		os.Remove(staged)
		Warn(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.ImportArchiveConfirm(manifest, token))
}

// CommitImport recreates the case from the staged archive and restores its
// binaries.
func (h *Handler) CommitImport(w http.ResponseWriter, r *http.Request, token string) {
	staged := stagedArchivePath(token)
	defer os.Remove(staged)

	zr, err := zip.OpenReader(staged)
	if err != nil {
		Warn(w, r, fmt.Errorf("import session expired, please upload the archive again: %w", err))
		return
	}
	defer zr.Close()

	manifest, err := readManifest(&zr.Reader)
	if err != nil {
		Warn(w, r, err)
		return
	}
	if err := validateManifest(manifest); err != nil {
		Warn(w, r, err)
		return
	}
	if err := validateArchivePaths(&zr.Reader); err != nil {
		Warn(w, r, err)
		return
	}
	arch, err := readCaseArchive(&zr.Reader)
	if err != nil {
		Warn(w, r, err)
		return
	}

	// recreate the case + every child record in one transaction; any collision
	// or constraint error rolls back, leaving no partial case
	if err := h.Store.ImportCaseArchive(arch); err != nil {
		Warn(w, r, err)
		return
	}

	// filesystem writes are not part of the DB transaction; do them last so a DB
	// failure never strands files on disk
	if err := restoreBinaries(&zr.Reader, arch.Case.ID, arch.Evidences); err != nil {
		Err(w, r, err)
		return
	}

	// grant the importing user access to what they just imported (administrators
	// keep wildcard access regardless)
	user := GetUser(r)
	if err := h.ACL.SaveCasePermissions(arch.Case.ID, []string{user.ID}); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/cases/"+arch.Case.ID+"/summary/", http.StatusSeeOther)
}

// stagedArchivePath returns the on-disk location for an upload staged under the
// given token. filepath.Base defends against a tampered token.
func stagedArchivePath(token string) string {
	return filepath.Join("files", "tmp", filepath.Base(token)+".zip")
}

func readManifest(zr *zip.Reader) (model.Manifest, error) {
	f, err := zr.Open("manifest.json")
	if err != nil {
		return model.Manifest{}, fmt.Errorf("archive is missing manifest.json: %w", err)
	}
	defer f.Close()

	var m model.Manifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return model.Manifest{}, fmt.Errorf("invalid manifest.json: %w", err)
	}
	return m, nil
}

func readCaseArchive(zr *zip.Reader) (model.CaseArchive, error) {
	f, err := zr.Open("case.json")
	if err != nil {
		return model.CaseArchive{}, fmt.Errorf("archive is missing case.json: %w", err)
	}
	defer f.Close()

	var arch model.CaseArchive
	if err := json.NewDecoder(f).Decode(&arch); err != nil {
		return model.CaseArchive{}, fmt.Errorf("invalid case.json: %w", err)
	}

	// the case id is user-controlled and used verbatim as a path component in
	// restoreBinaries; an alphanumeric id (the only form fp.Random ever mints)
	// cannot contain a separator and so cannot escape the files/ tree
	if !reAlnum.MatchString(arch.Case.ID) {
		return model.CaseArchive{}, fmt.Errorf("illegal case id in archive: %q", arch.Case.ID)
	}
	return arch, nil
}

func validateManifest(m model.Manifest) error {
	if m.Format != model.ArchiveFormat {
		return fmt.Errorf("not a dagobert case archive (format %q)", m.Format)
	}
	if m.FormatVersion != model.ArchiveFormatVersion {
		return fmt.Errorf("unsupported archive format version %d; this build supports version %d", m.FormatVersion, model.ArchiveFormatVersion)
	}
	cur, err := model.SchemaVersion()
	if err != nil {
		return err
	}
	if m.SchemaVersion != cur {
		return fmt.Errorf("archive schema version %d does not match this instance (%d); cross-version import is not supported", m.SchemaVersion, cur)
	}
	return nil
}

// validateArchivePaths rejects the whole archive if any evidence/malware entry
// name would escape its target directory (zip-slip). Entry names originate on a
// foreign instance and are untrusted; legitimate names are always a single flat
// path element.
func validateArchivePaths(zr *zip.Reader) error {
	for _, f := range zr.File {
		for _, prefix := range []string{"evidences/", "malware/"} {
			if _, ok, err := archiveBinaryName(prefix, f.Name); err != nil {
				return err
			} else if ok {
				break
			}
		}
	}
	return nil
}

// archiveBinaryName extracts the flat file name of a binary entry under prefix.
// It returns ok=false for entries that do not belong to the prefix (and for the
// bare directory entry), and an error for any name that is not a single safe
// path element.
func archiveBinaryName(prefix, name string) (string, bool, error) {
	if len(name) <= len(prefix) || name[:len(prefix)] != prefix {
		return "", false, nil
	}
	rel := name[len(prefix):]
	if rel != filepath.Base(rel) || rel == "." || rel == ".." {
		return "", true, fmt.Errorf("illegal entry path in archive: %q", name)
	}
	return rel, true, nil
}

// restoreBinaries writes the evidence files and malware samples from the archive
// to disk under the (preserved) case id. Stored evidence hashes are re-verified;
// a mismatch is a warning, not a failure (the metadata still imported).
func restoreBinaries(zr *zip.Reader, caseID string, evidences []model.Evidence) error {
	evHash := fp.ToMap(evidences, func(e model.Evidence) string { return e.Name })

	// shared decompressed-size budget across every binary in the archive, so a
	// bomb split over many entries can't slip past a per-file check
	budget := maxArchiveContentSize()

	for _, f := range zr.File {
		if rel, ok, err := archiveBinaryName("evidences/", f.Name); err != nil {
			return err
		} else if ok {
			sum, err := writeZipFile(f, filepath.Join("files", "evidences", caseID, rel), &budget)
			if err != nil {
				return err
			}
			if want := evHash[rel].Hash; want != "" && want != sum {
				slog.Warn("evidence hash mismatch on import", "case", caseID, "name", rel, "want", want, "got", sum)
			}
			continue
		}

		if rel, ok, err := archiveBinaryName("malware/", f.Name); err != nil {
			return err
		} else if ok {
			if _, err := writeZipFile(f, filepath.Join("files", "malware", caseID, rel), &budget); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeZipFile copies a single zip entry to dst, returning the SHA-1 of the
// written bytes. *budget is the number of decompressed bytes still allowed
// across the import; it is decremented by the amount written, and exceeding it
// is a hard error so a zip bomb can't exhaust the disk.
func writeZipFile(f *zip.File, dst string, budget *int64) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "", err
	}

	fr, err := f.Open()
	if err != nil {
		return "", err
	}
	defer fr.Close()

	fw, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}
	defer fw.Close()

	// read one byte past the budget so a copy that stops exactly at *budget is
	// distinguishable from one that would have overrun; the latter is rejected
	// before the truncated file is mistaken for a complete one.
	hasher := sha1.New()
	n, err := io.Copy(io.MultiWriter(fw, hasher), io.LimitReader(fr, *budget+1))
	if err != nil {
		return "", err
	}
	if n > *budget {
		return "", fmt.Errorf("archive exceeds maximum decompressed size of %d bytes", maxArchiveContentSize())
	}
	*budget -= n

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
