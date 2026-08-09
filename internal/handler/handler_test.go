package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newCSVImportRequest builds the multipart POST ImportCSV expects: a "file"
// field carrying the CSV body, addressed at the given case.
func newCSVImportRequest(t *testing.T, cid, body string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "import.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPost, "/cases/"+cid+"/events/import/csv", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.SetPathValue("cid", cid)
	return r
}

func TestImportCSVCollectsRowErrorsAndRollsBackOnFailure(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	before, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}

	body := "ID,Time,Type,Assets,Indicators,Event,Raw,Source,Custom\n" +
		",not-a-time,Other,,,first,,,\n" + // row 2: bad
		",2026-01-01T00:00:00Z,Other,,,second,,,\n" + // row 3: good
		",also-not-a-time,Other,,,third,,,\n" // row 4: bad

	h := &Handler{Store: db}
	r := newCSVImportRequest(t, kase.ID, body)
	rec := httptest.NewRecorder()
	h.EventImportCSV(rec, r)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	got := rec.Body.String()
	if !strings.Contains(got, "Row 2:") {
		t.Errorf("response missing Row 2 error: %s", got)
	}
	if !strings.Contains(got, "Row 4:") {
		t.Errorf("response missing Row 4 error: %s", got)
	}

	after, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("event count changed from %d to %d — a failed import must not commit anything", len(before), len(after))
	}
}

func TestImportCSVCommitsAndRedirectsOnSuccess(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	before, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}

	body := "ID,Time,Type,Assets,Indicators,Event,Raw,Source,Custom\n" +
		",2026-01-01T00:00:00Z,Other,,,first,,,\n" +
		",2026-01-02T00:00:00Z,Other,,,second,,,\n"

	h := &Handler{Store: db}
	r := newCSVImportRequest(t, kase.ID, body)
	rec := httptest.NewRecorder()
	h.EventImportCSV(rec, r)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	wantURI := "/cases/" + kase.ID + "/events/"
	if got := rec.Header().Get("Location"); got != wantURI {
		t.Errorf("redirect location = %q, want %q", got, wantURI)
	}

	after, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+2 {
		t.Errorf("event count = %d, want %d", len(after), len(before)+2)
	}
}

func TestImportCSVCollectsStructuralRowErrorsInsteadOfAborting(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	before, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}

	// row 2 has one field too many — wrong column count, not a business-logic
	// error — but the reader must keep going and collect row 3's failure too.
	body := "ID,Time,Type,Assets,Indicators,Event,Raw,Source,Custom\n" +
		",2026-01-01T00:00:00Z,Other,,,first,,,,extra\n" +
		",not-a-time,Other,,,second,,,\n"

	h := &Handler{Store: db}
	r := newCSVImportRequest(t, kase.ID, body)
	rec := httptest.NewRecorder()
	h.EventImportCSV(rec, r)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	got := rec.Body.String()
	if !strings.Contains(got, "Row 2:") {
		t.Errorf("response missing structural Row 2 error: %s", got)
	}
	if !strings.Contains(got, "Row 3:") {
		t.Errorf("response missing Row 3 error — reader aborted instead of collecting: %s", got)
	}

	after, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("event count changed from %d to %d — a failed import must not commit anything", len(before), len(after))
	}
}
