package stix

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var testTime = time.Date(2026, 6, 19, 8, 30, 0, 0, time.UTC)

func TestNewBundle(t *testing.T) {
	b := NewBundle()
	if b.Type != "bundle" || !strings.HasPrefix(b.ID, "bundle--") {
		t.Errorf("bad bundle envelope: %+v", b)
	}
	if len(b.Objects) != 0 {
		t.Errorf("new bundle should be empty, got %d objects", len(b.Objects))
	}
}

func TestAddIndicator_SetsRequiredProperties(t *testing.T) {
	b := NewBundle()
	b.AddIndicator("[ipv4-addr:value='198.51.100.7']", testTime)

	if len(b.Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(b.Objects))
	}
	obj := b.Objects[0]
	if obj.Type != "indicator" {
		t.Errorf("type = %q, want indicator", obj.Type)
	}
	if obj.SpecVersion != "2.1" {
		t.Errorf("spec_version = %q, want 2.1", obj.SpecVersion)
	}
	if !strings.HasPrefix(obj.ID, "indicator--") {
		t.Errorf("id must be present and prefixed: %q", obj.ID)
	}
	if obj.Created == "" || obj.Modified == "" {
		t.Errorf("created/modified are required: created=%q modified=%q", obj.Created, obj.Modified)
	}
	if obj.PatternType != "stix" {
		t.Errorf("pattern_type = %q, want stix", obj.PatternType)
	}
}

func TestAddIndicator_GeneratesUniqueIDs(t *testing.T) {
	b := NewBundle()
	b.AddIndicator("[ipv4-addr:value='198.51.100.7']", testTime)
	b.AddIndicator("[ipv4-addr:value='198.51.100.8']", testTime)
	if b.Objects[0].ID == b.Objects[1].ID {
		t.Errorf("indicator ids must be unique, both were %q", b.Objects[0].ID)
	}
}

func TestTimestamp_UTCWithZ(t *testing.T) {
	// Even with a non-UTC input the output must be UTC ("Z"-terminated).
	loc := time.FixedZone("CEST", 2*3600)
	got := Timestamp(testTime.In(loc))
	if !strings.HasSuffix(got, "Z") {
		t.Errorf("must be UTC/Z-terminated, got %q", got)
	}
	if _, err := time.Parse(time.RFC3339, got); err != nil {
		t.Errorf("not a valid RFC3339 timestamp: %q (%v)", got, err)
	}
	if got != "2026-06-19T08:30:00.000Z" {
		t.Errorf("timestamp normalization wrong: %q", got)
	}
}

func TestQuoteLiteral(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"O'Brien", `O\'Brien`},
		{`a\b`, `a\\b`},
		{`O'Brien\Svc`, `O\'Brien\\Svc`},
	}
	for _, tc := range tests {
		if got := QuoteLiteral(tc.in); got != tc.want {
			t.Errorf("QuoteLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBundle_IsValidJSON(t *testing.T) {
	b := NewBundle()
	b.AddIndicator("[ipv4-addr:value='198.51.100.7']", testTime)
	b.AddIndicator("[domain-name:value='evil.example.com']", testTime)

	out, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var rt Bundle
	if err := json.Unmarshal(out, &rt); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if len(rt.Objects) != 2 {
		t.Errorf("expected 2 objects after round-trip, got %d", len(rt.Objects))
	}
	// Required STIX property names must be present in the serialized form.
	s := string(out)
	for _, want := range []string{`"spec_version"`, `"created"`, `"modified"`, `"valid_from"`, `"pattern_type"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing property %s in output:\n%s", want, s)
		}
	}
}
