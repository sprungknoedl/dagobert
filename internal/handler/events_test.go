package handler

import (
	"testing"
	"time"

	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

func TestSaveTimesketchEventsDedupsOnReimport(t *testing.T) {
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

func TestSaveTimesketchIndicatorsMapsTypes(t *testing.T) {
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
