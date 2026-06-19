//go:build validate

// Built only under the "validate" tag (see `make validate-exports`). Validates a
// representative STIX 2.1 bundle, built via this package's API, against the
// official STIX 2.1 JSON schemas using stix2-validator. The validator binary may
// be supplied via the STIX2_VALIDATOR environment variable; otherwise it is
// looked up on PATH.

package stix

import (
	"encoding/json"
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

// representative builds a bundle covering every STIX pattern shape the exporters
// produce: quoted hash keys, an object-extension path, a custom object type, an
// AND-combined observation, and a string literal containing an escaped quote.
func representative() *Bundle {
	now := time.Now()
	b := NewBundle()
	for _, pattern := range []string{
		"[ipv4-addr:value='198.51.100.7']",
		"[domain-name:value='evil.example.com']",
		"[url:value='http://evil.example.com/a?b=1&c=2']",
		"[directory:path='/opt/evil' AND file:name='run.sh']",
		"[file:hashes.MD5='" + strings.Repeat("a", 32) + "']",
		"[file:hashes.'SHA-1'='" + strings.Repeat("b", 40) + "']",
		"[file:hashes.'SHA-256'='" + strings.Repeat("c", 64) + "']",
		"[file:hashes.Other='deadbeef']",
		"[process:extensions.'windows-service-ext'.service_name='Evil & Co']",
		"[x-example:value='" + QuoteLiteral("O'Brien") + "']",
	} {
		b.AddIndicator(pattern, now)
	}
	return b
}

func TestValidateAgainstSchemas(t *testing.T) {
	bin := lookupTool(t, "STIX2_VALIDATOR", "stix2_validator")

	out, err := json.MarshalIndent(representative(), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// stix2_validator only discovers files with a .json extension.
	file := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(file, out, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := exec.Command(bin, file).CombinedOutput()
	if err != nil {
		t.Fatalf("STIX bundle failed validation:\n%s\n--- document ---\n%s", got, out)
	}
	t.Logf("stix2_validator: %s", strings.TrimSpace(string(got)))
}
