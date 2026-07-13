package handler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
)

// sha1 of "hello world"
const helloHash = "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"

func TestResolveEvidenceFileAdoptsFileOnDisk(t *testing.T) {
	t.Chdir(t.TempDir())
	dir := filepath.Join("files", "evidences", "case01")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dump.bin"), []byte("hello world"), 0666); err != nil {
		t.Fatal(err)
	}

	dto := model.Evidence{ID: "ev01", CaseID: "case01", Name: "dump.bin", Fileless: true}
	got, attached, details, err := resolveEvidenceFile(dto, model.Evidence{}, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !attached || details == "" {
		t.Errorf("got attached=%v details=%q, want an attach with details", attached, details)
	}
	if got.Fileless || got.Size != 11 || got.Hash != helloHash {
		t.Errorf("got size=%d hash=%q fileless=%v, want size=11 hash=%q fileless=false", got.Size, got.Hash, got.Fileless, helloHash)
	}
}

func TestResolveEvidenceFileNoFileIsNotAnError(t *testing.T) {
	t.Chdir(t.TempDir())
	dto := model.Evidence{ID: "ev01", CaseID: "case01", Name: "missing.bin", Fileless: true}
	got, attached, _, err := resolveEvidenceFile(dto, model.Evidence{}, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if attached {
		t.Error("want not attached when nothing to adopt")
	}
	if !got.Fileless || got.Size != 0 || got.Hash != "" {
		t.Errorf("got size=%d hash=%q fileless=%v, want empty metadata and still fileless", got.Size, got.Hash, got.Fileless)
	}
}

func TestResolveEvidenceFileKeepsStoredMetadata(t *testing.T) {
	t.Chdir(t.TempDir())
	old := model.Evidence{ID: "ev01", CaseID: "case01", Name: "dump.bin", Size: 11, Hash: helloHash}

	// simulate a form save of the existing entry without a new upload
	dto := model.Evidence{ID: old.ID, CaseID: old.CaseID, Name: old.Name}
	got, attached, _, err := resolveEvidenceFile(dto, old, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if attached {
		t.Error("want not attached for a file-backed entry")
	}
	if got.Size != 11 || got.Hash != helloHash {
		t.Errorf("got size=%d hash=%q, want stored metadata", got.Size, got.Hash)
	}
}
