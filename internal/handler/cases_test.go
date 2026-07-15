package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCaseDeleteRemovesCaseFiles(t *testing.T) {
	db := setupArchiveDB(t)
	seedCase(t, db)
	t.Chdir(t.TempDir())

	evidenceDir := filepath.Join("files", "evidences", "case01")
	malwareDir := filepath.Join("files", "malware", "case01")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(malwareDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "dummy.txt"), []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(malwareDir, "dummy.zip"), []byte("malware"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db}
	r := httptest.NewRequest(http.MethodDelete, "/cases/case01?confirm=yes", nil)
	r.SetPathValue("cid", "case01")
	rec := httptest.NewRecorder()

	h.CaseDelete(rec, r)

	if _, err := os.Stat(evidenceDir); !os.IsNotExist(err) {
		t.Errorf("evidence dir still exists: %v", err)
	}
	if _, err := os.Stat(malwareDir); !os.IsNotExist(err) {
		t.Errorf("malware dir still exists: %v", err)
	}
	if _, err := db.GetCase("case01"); err == nil {
		t.Errorf("case still exists after delete")
	}
}
