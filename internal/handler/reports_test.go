package handler

import (
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
)

func TestLoadTemplateRejectsTraversal(t *testing.T) {
	// names that escape files/templates/ must be rejected by the path guard,
	// before any filesystem access, with the generic "invalid template" error.
	for _, name := range []string{
		"../../evidences/case02/secret.odt",
		"../report.odt",
		"sub/report.odt",
		`..\report.odt`, // backslash: a path separator on Windows
		`sub\report.odt`,
		".",
		"..",
	} {
		_, err := LoadTemplate(name)
		if err == nil || err.Error() != "invalid template" {
			t.Errorf("LoadTemplate(%q) = %v, want \"invalid template\"", name, err)
		}
	}
}

func TestResolveReportFileRejectsInvalidUpload(t *testing.T) {
	t.Chdir(t.TempDir())
	f, size := openUpload(t, []byte("not a real docx"))

	dto := model.ReportTemplate{ID: "new", Name: "broken.docx"}
	err := resolveReportTemplateFile(nil, dto, true, f, &multipart.FileHeader{Filename: "broken.docx", Size: size})

	var terr templateError
	if !errors.As(err, &terr) {
		t.Fatalf("got %v, want templateError", err)
	}
	// a rejected brand-new upload must not leave the broken file behind
	if _, err := os.Stat(filepath.Join("files", "templates", "broken.docx")); !os.IsNotExist(err) {
		t.Error("rejected template was left on disk")
	}
}

func TestResolveReportFileRenamesOnNameChange(t *testing.T) {
	t.Chdir(t.TempDir())
	db := setupArchiveDB(t)
	obj := model.ReportTemplate{ID: "rep01", Name: "old.docx"}
	if err := db.SaveReportTemplate(obj); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("files", "templates")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "old.docx"), []byte("tmpl"), 0666)

	dto := model.ReportTemplate{ID: "rep01", Name: "new.docx"}
	if err := resolveReportTemplateFile(db, dto, false, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.docx")); err != nil {
		t.Error("stored template was not renamed")
	}
}
