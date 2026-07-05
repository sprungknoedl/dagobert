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

	dto := model.Evidence{ID: "new", CaseID: "case01", Name: "dump.bin"}
	got, err := resolveEvidenceFile(nil, dto, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != 11 || got.Hash != helloHash {
		t.Errorf("got size=%d hash=%q, want size=11 hash=%q", got.Size, got.Hash, helloHash)
	}
}

func TestResolveEvidenceFileNoFileIsNotAnError(t *testing.T) {
	t.Chdir(t.TempDir())
	dto := model.Evidence{ID: "new", CaseID: "case01", Name: "missing.bin"}
	got, err := resolveEvidenceFile(nil, dto, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != 0 || got.Hash != "" {
		t.Errorf("got size=%d hash=%q, want empty metadata", got.Size, got.Hash)
	}
}

func TestResolveEvidenceFileKeepsStoredMetadata(t *testing.T) {
	t.Chdir(t.TempDir())
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	ev := model.Evidence{ID: "ev01", CaseID: kase.ID, Name: "dump.bin", Size: 11, Hash: helloHash}
	if err := db.SaveEvidence(kase.ID, ev); err != nil {
		t.Fatal(err)
	}

	// simulate a form save of the existing entry without a new upload
	dto := model.Evidence{ID: ev.ID, CaseID: kase.ID, Name: ev.Name, Size: ev.Size}
	got, err := resolveEvidenceFile(db, dto, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != 11 || got.Hash != helloHash {
		t.Errorf("got size=%d hash=%q, want stored metadata", got.Size, got.Hash)
	}
}
