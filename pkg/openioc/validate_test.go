//go:build validate

// Built only under the "validate" tag (see `make validate-exports`). Validates
// a representative OpenIOC document, built via this package's API, against the
// vendored OpenIOC 1.1 XSD using xmllint. The xmllint binary may be supplied via
// the XMLLINT environment variable; otherwise it is looked up on PATH.

package openioc

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lookupTool(t *testing.T, env, name string) string {
	t.Helper()
	if p := os.Getenv(env); p != "" {
		return p
	}
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not found (set %s or install it); skipping", name, env)
	}
	return p
}

// representative builds a document exercising every item shape the exporters
// produce, including a value containing XML metacharacters.
func representative() *Document {
	doc := New("Validator", time.Now())
	doc.AddItem("is", Context{Document: "PortItem", Search: "PortItem/RemoteIP", Type: "mir"}, "IP", "198.51.100.7")
	doc.AddItem("contains", Context{Document: "DnsEntryItem", Search: "DnsEntryItem/Host", Type: "mir"}, "string", "evil.example.com")
	doc.AddItem("contains", Context{Document: "FileItem", Search: "FileItem/FileFullPath", Type: "mir"}, "string", `C:\Temp\a & b<c>.exe`)
	doc.AddItem("is", Context{Document: "FileItem", Search: "FileItem/Md5sum", Type: "mir"}, "string", strings.Repeat("a", 32))
	doc.AddItem("is", Context{Document: "ServiceItem", Search: "ServiceItem/Name", Type: "mir"}, "string", "Evil & Co")
	return doc
}

func TestValidateAgainstXSD(t *testing.T) {
	bin := lookupTool(t, "XMLLINT", "xmllint")

	out, err := xml.MarshalIndent(representative(), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	file := filepath.Join(t.TempDir(), "doc.ioc")
	if err := os.WriteFile(file, out, 0o644); err != nil {
		t.Fatal(err)
	}

	xsd := filepath.Join("testdata", "ioc.xsd")
	got, err := exec.Command(bin, "--noout", "--schema", xsd, file).CombinedOutput()
	if err != nil {
		t.Fatalf("OpenIOC document failed XSD validation:\n%s\n--- document ---\n%s", got, out)
	}
	t.Logf("xmllint: %s", strings.TrimSpace(string(got)))
}
