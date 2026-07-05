package handler

import (
	"strings"
	"testing"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
)

var exportTestTime = time.Date(2026, 6, 19, 8, 30, 0, 0, time.UTC)

// These tests cover the Dagobert-specific indicator-type mapping. Format-level
// validity (required fields, escaping, timestamps) is tested in pkg/openioc and
// pkg/stix.

func TestBuildOpenIOC_TypeMapping(t *testing.T) {
	tests := []struct {
		name       string
		ioc        model.Indicator
		wantSearch string
		wantType   string
		wantCond   string
	}{
		{"ip", model.Indicator{Type: "IP", Value: "198.51.100.7"}, "PortItem/RemoteIP", "IP", "is"},
		{"domain", model.Indicator{Type: "Domain", Value: "evil.example.com"}, "DnsEntryItem/Host", "string", "contains"},
		{"url", model.Indicator{Type: "URL", Value: "http://evil.example.com/x"}, "Network/URI", "string", "contains"},
		{"path", model.Indicator{Type: "Path", Value: `C:\Windows\evil.exe`}, "FileItem/FileFullPath", "string", "contains"},
		{"md5", model.Indicator{Type: "Hash", Value: strings.Repeat("a", 32)}, "FileItem/Md5sum", "string", "is"},
		{"sha1", model.Indicator{Type: "Hash", Value: strings.Repeat("b", 40)}, "FileItem/Sha1sum", "string", "is"},
		{"sha256", model.Indicator{Type: "Hash", Value: strings.Repeat("c", 64)}, "FileItem/Sha256sum", "string", "is"},
		{"unknown-hash", model.Indicator{Type: "Hash", Value: "deadbeef"}, "FileItem/Other", "string", "is"},
		{"service", model.Indicator{Type: "Service", Value: "EvilSvc"}, "ServiceItem/Name", "string", "is"},
		{"other", model.Indicator{Type: "Other", Value: "whatever"}, "Other/Other", "string", "is"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := buildOpenIOC([]model.Indicator{tc.ioc}, "Alice", exportTestTime)
			items := doc.Criteria[0].Items
			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
			it := items[0]
			if it.Condition != tc.wantCond {
				t.Errorf("condition = %q, want %q", it.Condition, tc.wantCond)
			}
			if it.Context.Search != tc.wantSearch {
				t.Errorf("search = %q, want %q", it.Context.Search, tc.wantSearch)
			}
			if it.Content.Type != tc.wantType {
				t.Errorf("content type = %q, want %q", it.Content.Type, tc.wantType)
			}
			if it.Content.Value != tc.ioc.Value {
				t.Errorf("content value = %q, want %q", it.Content.Value, tc.ioc.Value)
			}
		})
	}
}

func TestBuildOpenIOC_IncludeUnknownType(t *testing.T) {
	doc := buildOpenIOC([]model.Indicator{{Type: "Bogus", Value: "x"}}, "Alice", exportTestTime)
	if len(doc.Criteria[0].Items) != 1 {
		t.Errorf("unknown type shouldn't be skipped, got %d items", len(doc.Criteria[0].Items))
	}
}

func TestBuildStixBundle_PatternMapping(t *testing.T) {
	tests := []struct {
		name        string
		ioc         model.Indicator
		wantPattern string
	}{
		{"ip", model.Indicator{Type: "IP", Value: "198.51.100.7"}, "[ipv4-addr:value='198.51.100.7']"},
		{"domain", model.Indicator{Type: "Domain", Value: "evil.example.com"}, "[domain-name:value='evil.example.com']"},
		{"url", model.Indicator{Type: "URL", Value: "http://evil.example.com/x"}, "[url:value='http://evil.example.com/x']"},
		{"path", model.Indicator{Type: "Path", Value: "/opt/evil/run.sh"}, "[directory:path='/opt/evil' AND file:name='run.sh']"},
		{"md5", model.Indicator{Type: "Hash", Value: strings.Repeat("a", 32)}, "[file:hashes.MD5='" + strings.Repeat("a", 32) + "']"},
		{"sha1", model.Indicator{Type: "Hash", Value: strings.Repeat("b", 40)}, "[file:hashes.'SHA-1'='" + strings.Repeat("b", 40) + "']"},
		{"sha256", model.Indicator{Type: "Hash", Value: strings.Repeat("c", 64)}, "[file:hashes.'SHA-256'='" + strings.Repeat("c", 64) + "']"},
		{"service", model.Indicator{Type: "Service", Value: "EvilSvc"}, "[process:extensions.'windows-service-ext'.service_name='EvilSvc']"},
		{"other", model.Indicator{Type: "Other", Value: "whatever"}, "[x-dagobert:value='whatever']"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := buildStixBundle([]model.Indicator{tc.ioc}, exportTestTime)
			if len(b.Objects) != 1 {
				t.Fatalf("expected 1 object, got %d", len(b.Objects))
			}
			if got := b.Objects[0].Pattern; got != tc.wantPattern {
				t.Errorf("pattern = %q, want %q", got, tc.wantPattern)
			}
		})
	}
}

func TestBuildStixBundle_EscapesQuotesInPattern(t *testing.T) {
	b := buildStixBundle([]model.Indicator{{Type: "Service", Value: `O'Brien\Svc`}}, exportTestTime)
	want := `[process:extensions.'windows-service-ext'.service_name='O\'Brien\\Svc']`
	if got := b.Objects[0].Pattern; got != want {
		t.Errorf("pattern = %q, want %q", got, want)
	}
}

func TestBuildStixBundle_IncludeUnknownType(t *testing.T) {
	b := buildStixBundle([]model.Indicator{{Type: "Bogus", Value: "x"}}, exportTestTime)
	if len(b.Objects) != 1 {
		t.Errorf("unknown type shouldn't be skipped, got %d objects", len(b.Objects))
	}
}
