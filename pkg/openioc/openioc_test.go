package openioc

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

var testTime = time.Date(2026, 6, 19, 8, 30, 0, 0, time.UTC)

func TestNew_DocumentMetadata(t *testing.T) {
	doc := New("Alice", testTime)

	if doc.Namespace != Namespace {
		t.Errorf("missing/wrong xmlns: %q", doc.Namespace)
	}
	if doc.ID == "" {
		t.Error("root id must not be empty")
	}
	if doc.LastModified.IsZero() || doc.PublishedDate.IsZero() {
		t.Error("last-modified / published-date must be set")
	}
	if len(doc.Criteria) != 1 || doc.Criteria[0].Operator != "OR" || doc.Criteria[0].ID == "" {
		t.Errorf("expected one OR Indicator with id, got %+v", doc.Criteria)
	}
	if doc.Metadata.AuthoredBy != "Alice" {
		t.Errorf("authored_by = %q, want Alice", doc.Metadata.AuthoredBy)
	}
}

func TestAddItem_GeneratesUniqueIDs(t *testing.T) {
	doc := New("Alice", testTime)
	doc.AddItem("is", Context{Document: "PortItem"}, "IP", "198.51.100.7")
	doc.AddItem("is", Context{Document: "PortItem"}, "IP", "198.51.100.8")

	items := doc.Criteria[0].Items
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID == "" || items[0].ID == items[1].ID {
		t.Errorf("item ids must be present and unique: %q, %q", items[0].ID, items[1].ID)
	}
}

func TestDocument_IsWellFormedAndUsesSchemaElementNames(t *testing.T) {
	doc := New("Alice", testTime)
	doc.AddItem("contains", Context{Document: "DnsEntryItem", Search: "DnsEntryItem/Host", Type: "mir"}, "string", "evil.example.com")

	out, err := xml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Re-parse: catches non-well-formed output.
	var rt Document
	if err := xml.Unmarshal(out, &rt); err != nil {
		t.Fatalf("output is not well-formed XML: %v", err)
	}

	s := string(out)
	for _, want := range []string{"<authored_by>", "<authored_date>", "<criteria>", "<IndicatorItem", "<Context", "<Content"} {
		if !strings.Contains(s, want) {
			t.Errorf("expected element %q in output:\n%s", want, s)
		}
	}
	// snake_case schema names, not Go field names.
	if strings.Contains(s, "<AuthoredBy>") {
		t.Errorf("metadata must use schema element names, got Go field names:\n%s", s)
	}
}

func TestDocument_EscapesSpecialCharacters(t *testing.T) {
	// A value with XML metacharacters must not break the document.
	val := `C:\Temp\a & b<c>.exe`
	doc := New("Alice", testTime)
	doc.AddItem("contains", Context{Document: "FileItem", Search: "FileItem/FileFullPath", Type: "mir"}, "string", val)

	out, err := xml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "& b<c>") {
		t.Errorf("special characters must be escaped, found raw value in:\n%s", out)
	}

	var rt Document
	if err := xml.Unmarshal(out, &rt); err != nil {
		t.Fatalf("output not well-formed: %v", err)
	}
	if got := rt.Criteria[0].Items[0].Content.Value; got != val {
		t.Errorf("round-trip value mismatch: got %q want %q", got, val)
	}
}
