package handler

import (
	"testing"
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
