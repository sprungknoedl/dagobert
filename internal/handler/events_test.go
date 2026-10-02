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

	got, err := db.GetEvent(kase.ID, "_ts_"+kase.ID+"_ts42")
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

// TestSaveTimesketchEventsLinks re-imports an event an analyst has linked to an
// asset and indicator, and checks the re-import keeps those links.
func TestSaveTimesketchEventsLinks(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)

	ev := timesketch.Event{ID: "ts42", Datetime: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), Message: "logon"}
	if err := saveTimesketchEvents(db, kase.ID, []timesketch.Event{ev}); err != nil {
		t.Fatal(err)
	}

	id := "_ts_" + kase.ID + "_ts42"
	got, err := db.GetEvent(kase.ID, id)
	if err != nil {
		t.Fatal(err)
	}
	got.Assets = []model.Asset{{ID: "asset01"}}
	got.Indicators = []model.Indicator{{ID: "ind01"}}
	if err := db.SaveEvent(kase.ID, got, true); err != nil {
		t.Fatal(err)
	}

	if err := saveTimesketchEvents(db, kase.ID, []timesketch.Event{ev}); err != nil {
		t.Fatal(err)
	}

	got, err = db.GetEvent(kase.ID, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Assets) != 1 || len(got.Indicators) != 1 {
		t.Errorf("re-import dropped links: assets %v, indicators %v", got.Assets, got.Indicators)
	}
}

// TestSaveTimesketchEventsCrossCase imports the same Timesketch event into two
// cases, which must not collide on the event ID.
func TestSaveTimesketchEventsCrossCase(t *testing.T) {
	db := setupArchiveDB(t)
	a := seedCase(t, db)
	b := model.Case{ID: "case02", Name: "Operation Other"}
	if err := db.SaveCase(b); err != nil {
		t.Fatal(err)
	}

	ev := timesketch.Event{ID: "ts42", Datetime: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), Message: "logon"}
	for _, cid := range []string{a.ID, b.ID} {
		if err := saveTimesketchEvents(db, cid, []timesketch.Event{ev}); err != nil {
			t.Fatalf("import into %s: %v", cid, err)
		}
	}

	list, err := db.ListEvents(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("got %d events in second case, want 1", len(list))
	}
}

// TestSaveTimesketchIndicatorsTwice re-imports the same intelligence, which
// must keep the existing indicators instead of failing on the unique index.
func TestSaveTimesketchIndicatorsTwice(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)

	values := []timesketch.Intelligence{{IOC: "evil.example", Type: "hostname"}}
	for range 2 {
		if err := saveTimesketchIndicators(db, kase.ID, values); err != nil {
			t.Fatal(err)
		}
	}

	list, err := db.ListIndicators(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ind := range list {
		if ind.Value == "evil.example" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("got %d evil.example indicators, want 1", n)
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
