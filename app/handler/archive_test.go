package handler

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/app/model"
)

func setupArchiveDB(t *testing.T) *model.Store {
	t.Helper()
	db, err := model.Connect(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.RawConn.Close() })

	source, _ := iofs.New(model.Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	return db
}

// seedCase populates a case with linked child records and returns it.
func seedCase(t *testing.T, db *model.Store) model.Case {
	t.Helper()
	kase := model.Case{ID: "case01", Name: "Operation Test", Severity: "High", SketchID: 42}
	if err := db.SaveCase(kase); err != nil {
		t.Fatal(err)
	}

	asset := model.Asset{ID: "asset01", CaseID: kase.ID, Name: "DC01", Type: "Host", Status: "Compromised"}
	if err := db.SaveAsset(kase.ID, asset); err != nil {
		t.Fatal(err)
	}

	ind := model.Indicator{ID: "ind01", CaseID: kase.ID, Type: "IP", Value: "198.51.100.7", TLP: "TLP:RED"}
	if err := db.SaveIndicator(kase.ID, ind, false); err != nil {
		t.Fatal(err)
	}

	ev := model.Event{
		ID: "ev01", CaseID: kase.ID, Type: "Other", Event: "lateral movement",
		Assets:     []model.Asset{{ID: asset.ID}},
		Indicators: []model.Indicator{{ID: ind.ID}},
	}
	if err := db.SaveEvent(kase.ID, ev, false); err != nil {
		t.Fatal(err)
	}

	mal := model.Malware{ID: "mal01", CaseID: kase.ID, Path: "evil.exe", Hash: "abc123", Asset: model.Asset{ID: asset.ID}}
	if err := db.SaveMalware(kase.ID, mal); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveNote(kase.ID, model.Note{ID: "note01", CaseID: kase.ID, Title: "n1"}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveTask(kase.ID, model.Task{ID: "task01", CaseID: kase.ID, Task: "t1"}); err != nil {
		t.Fatal(err)
	}
	return kase
}

// exportToBuffer writes a metadata-only archive for cid into a buffer.
func exportToBuffer(t *testing.T, db *model.Store, cid string) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := writeCaseArchive(buf, db, cid, "https://src.example.com", "tester@example.com", false); err != nil {
		t.Fatal(err)
	}
	return buf
}

func openZip(t *testing.T, buf *bytes.Buffer) *zip.Reader {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}

func TestArchiveRoundTrip(t *testing.T) {
	src := setupArchiveDB(t)
	seedCase(t, src)

	buf := exportToBuffer(t, src, "case01")
	zr := openZip(t, buf)

	manifest, err := readManifest(zr)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateManifest(manifest); err != nil {
		t.Fatalf("manifest should be valid: %v", err)
	}
	if manifest.OriginalCaseID != "case01" || manifest.Counts["events"] != 1 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}

	arch, err := readCaseArchive(zr)
	if err != nil {
		t.Fatal(err)
	}

	dst := setupArchiveDB(t)
	if err := dst.ImportCaseArchive(arch); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// ids preserved
	kase, err := dst.GetCase("case01")
	if err != nil {
		t.Fatalf("case not imported: %v", err)
	}
	if kase.SketchID != 42 {
		t.Errorf("SketchID = %d, want 42", kase.SketchID)
	}

	// counts match
	assets, _ := dst.ListAssets("case01")
	indicators, _ := dst.ListIndicators("case01")
	events, _ := dst.ListEvents("case01")
	malware, _ := dst.ListMalware("case01")
	notes, _ := dst.ListNotes("case01")
	tasks, _ := dst.ListTasks("case01")
	if len(assets) != 1 || len(indicators) != 1 || len(events) != 1 || len(malware) != 1 || len(notes) != 1 || len(tasks) != 1 {
		t.Fatalf("counts mismatch: assets=%d ind=%d ev=%d mal=%d notes=%d tasks=%d",
			len(assets), len(indicators), len(events), len(malware), len(notes), len(tasks))
	}

	// links intact
	if len(events[0].Assets) != 1 || events[0].Assets[0].ID != "asset01" {
		t.Errorf("event->asset link lost: %+v", events[0].Assets)
	}
	if len(events[0].Indicators) != 1 || events[0].Indicators[0].ID != "ind01" {
		t.Errorf("event->indicator link lost: %+v", events[0].Indicators)
	}
	if malware[0].AssetID != "asset01" {
		t.Errorf("malware.AssetID = %q, want asset01", malware[0].AssetID)
	}
}

func TestArchiveRoundTripBinaries(t *testing.T) {
	// export and import run in separate working dirs to mimic two instances
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })

	content := []byte("disk image bytes")
	sum := fmt.Sprintf("%x", sha1.Sum(content))

	// --- export side ---
	exportDir := t.TempDir()
	if err := os.Chdir(exportDir); err != nil {
		t.Fatal(err)
	}
	src := setupArchiveDB(t)
	if err := src.SaveCase(model.Case{ID: "case01", Name: "c"}); err != nil {
		t.Fatal(err)
	}
	if err := src.SaveEvidence("case01", model.Evidence{ID: "evi01", CaseID: "case01", Name: "image.dd", Hash: sum, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join("files", "evidences", "case01"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("files", "evidences", "case01", "image.dd"), content, 0644); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := writeCaseArchive(buf, src, "case01", "https://src", "t@e", true); err != nil {
		t.Fatal(err)
	}

	// --- import side ---
	importDir := t.TempDir()
	if err := os.Chdir(importDir); err != nil {
		t.Fatal(err)
	}
	dst := setupArchiveDB(t)
	zr := openZip(t, buf)
	arch, err := readCaseArchive(zr)
	if err != nil {
		t.Fatal(err)
	}
	if err := dst.ImportCaseArchive(arch); err != nil {
		t.Fatal(err)
	}
	if err := restoreBinaries(zr, "case01", arch.Evidences); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(importDir, "files", "evidences", "case01", "image.dd"))
	if err != nil {
		t.Fatalf("evidence not restored: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("restored content mismatch")
	}
	if fmt.Sprintf("%x", sha1.Sum(got)) != sum {
		t.Errorf("restored hash mismatch")
	}
}

func TestArchiveDuplicateIDRejected(t *testing.T) {
	db := setupArchiveDB(t)
	seedCase(t, db)

	buf := exportToBuffer(t, db, "case01")
	arch, err := readCaseArchive(openZip(t, buf))
	if err != nil {
		t.Fatal(err)
	}

	// importing into a store that already holds the case must fail and leave the
	// existing case untouched
	if err := db.ImportCaseArchive(arch); err == nil {
		t.Fatal("expected duplicate-id error, got nil")
	}

	assets, _ := db.ListAssets("case01")
	if len(assets) != 1 {
		t.Errorf("existing case clobbered: %d assets", len(assets))
	}
}

func TestArchiveSchemaMismatchRejected(t *testing.T) {
	cur, err := model.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	m := model.Manifest{Format: model.ArchiveFormat, FormatVersion: model.ArchiveFormatVersion, SchemaVersion: cur + 1}
	if err := validateManifest(m); err == nil {
		t.Fatal("expected schema mismatch error, got nil")
	}
}

func TestArchiveZipSlipRejected(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	fw, _ := zw.Create("evidences/../../etc/passwd")
	fw.Write([]byte("pwned"))
	zw.Close()

	if err := validateArchivePaths(openZip(t, buf)); err == nil {
		t.Fatal("expected zip-slip rejection, got nil")
	}
}

func TestArchiveTraversalCaseIDRejected(t *testing.T) {
	for _, id := range []string{"../../tmp", "a/b", "a.b", "", `..\..\tmp`} {
		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		fw, _ := zw.Create("case.json")
		body, _ := json.Marshal(model.CaseArchive{Case: model.Case{ID: id, Name: "x"}})
		fw.Write(body)
		zw.Close()

		if _, err := readCaseArchive(openZip(t, buf)); err == nil {
			t.Errorf("expected rejection for case id %q, got nil", id)
		}
	}
}
