package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

func TestSaveTimesketchEvents(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)

	ev := timesketch.Event{
		ID:       "ts42",
		Datetime: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Message:  "first import",
		Source:   map[string]any{"message": "first import"},
	}
	if err := saveTimesketchEvents(db, kase.ID, []timesketch.Event{ev}); err != nil {
		t.Fatal(err)
	}

	// a re-import must not duplicate or clobber the (possibly analyst-edited) event
	ev.Message = "second import"
	if err := saveTimesketchEvents(db, kase.ID, []timesketch.Event{ev}); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetEvent(kase.ID, "_ts_ts42")
	if err != nil {
		t.Fatal(err)
	}
	if got.Event != "first import" {
		t.Errorf("re-import clobbered the existing event: got %q", got.Event)
	}
	if got.Source != "Timesketch" {
		t.Errorf("got source %q, want Timesketch", got.Source)
	}
}

func TestSaveTimesketchIndicators(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)

	values := []timesketch.Intelligence{
		{IOC: "198.51.100.99", Type: "ipv4"},
		{IOC: "evil.example", Type: "hostname"},
	}
	if err := saveTimesketchIndicators(db, kase.ID, values); err != nil {
		t.Fatal(err)
	}

	list, err := db.ListIndicators(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]string{}
	for _, ind := range list {
		types[ind.Value] = ind.Type
	}
	if types["198.51.100.99"] != "IP" || types["evil.example"] != "Domain" {
		t.Errorf("type mapping wrong: %v", types)
	}
}

// TestEventSaveJSON creates an event through the JSON API with assets and
// indicators as plain strings and custom attributes, and checks that the
// links and custom values are stored.
func TestEventSaveJSON(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	h := &Handler{Store: db}

	r := newJSONRequest(t, "/cases/"+kase.ID+"/events/new", map[string]any{
		"Time":       "2026-07-01T12:00:00Z",
		"Type":       "Execution",
		"Event":      "malware executed",
		"Assets":     []string{"DC01"},
		"Indicators": []string{"198.51.100.7"},
		"Custom":     map[string]string{"Ticket": "INC-1"},
	})
	r.SetPathValue("cid", kase.ID)
	r.SetPathValue("id", "new")
	rec := httptest.NewRecorder()
	h.EventSave(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var resp model.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetEvent(kase.ID, resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Event != "malware executed" {
		t.Errorf("got event %q, want malware executed", got.Event)
	}
	if len(got.Assets) != 1 || got.Assets[0].ID != "asset01" {
		t.Errorf("got assets %v, want [asset01]", got.Assets)
	}
	if len(got.Indicators) != 1 || got.Indicators[0].ID != "ind01" {
		t.Errorf("got indicators %v, want [ind01]", got.Indicators)
	}
	if got.Custom["Ticket"] != "INC-1" {
		t.Errorf("got custom %v, want Ticket=INC-1", got.Custom)
	}
}
