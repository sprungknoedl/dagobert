package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// flashToast returns the decoded value of the flash_toast cookie set on rec
// (unset if the cookie is absent or was cleared via a non-negative MaxAge).
func flashToast(t *testing.T, rec *httptest.ResponseRecorder) (string, bool) {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name != flashToastCookie {
			continue
		}
		if c.MaxAge < 0 {
			return "", false
		}
		msg, err := url.QueryUnescape(c.Value)
		if err != nil {
			t.Fatal(err)
		}
		return msg, true
	}
	return "", false
}

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
	if msg, ok := flashToast(t, rec); ok {
		t.Errorf("flash toast = %q, want unset — a failed import must not show a success toast", msg)
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
	wantToast := "Imported 2 records."
	if msg, ok := flashToast(t, rec); !ok || msg != wantToast {
		t.Errorf("flash toast = %q (set=%v), want %q", msg, ok, wantToast)
	}

	after, err := db.ListEvents(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+2 {
		t.Errorf("event count = %d, want %d", len(after), len(before)+2)
	}
}

func TestImportCSVSingularRecordToast(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)

	body := "ID,Time,Type,Assets,Indicators,Event,Raw,Source,Custom\n" +
		",2026-01-01T00:00:00Z,Other,,,first,,,\n"

	h := &Handler{Store: db}
	r := newCSVImportRequest(t, kase.ID, body)
	rec := httptest.NewRecorder()
	h.EventImportCSV(rec, r)

	wantToast := "Imported 1 record."
	if msg, ok := flashToast(t, rec); !ok || msg != wantToast {
		t.Errorf("flash toast = %q (set=%v), want %q", msg, ok, wantToast)
	}
}

func TestRedirectAfterSaveSetsSuccessToast(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/cases/case01/assets/new", nil)
	rec := httptest.NewRecorder()

	RedirectAfterSave(rec, r, "/cases/case01/assets/", nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	wantToast := "Saved."
	if msg, ok := flashToast(t, rec); !ok || msg != wantToast {
		t.Errorf("flash toast = %q (set=%v), want %q", msg, ok, wantToast)
	}
}

// TestFlashToastMiddlewareDeliversTheToastOnTheNextRequest is the piece that
// makes the redirect-toast handoff actually work end to end: a save handler
// stages a cookie via SetFlashToast, and it's the *next* request (the
// redirect's target) whose response needs X-Up-Accept-Layer — not the 303
// itself, which Unpoly's XHR follows internally without exposing its headers.
func TestFlashToastMiddlewareDeliversTheToastOnTheNextRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// first request: a save stages the toast, no downstream header yet
	save := httptest.NewRecorder()
	RedirectAfterSave(save, httptest.NewRequest(http.MethodPost, "/cases/case01/assets/new", nil), "/cases/case01/assets/", nil)
	if got := save.Header().Get("X-Up-Accept-Layer"); got != "" {
		t.Errorf("X-Up-Accept-Layer on the redirect itself = %q, want unset", got)
	}

	// second request: the redirect's target carries the cookie the first
	// response set; FlashToast must turn it into the header here
	follow := httptest.NewRequest(http.MethodGet, "/cases/case01/assets/", nil)
	for _, c := range save.Result().Cookies() {
		follow.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	FlashToast(next).ServeHTTP(rec, follow)

	wantHeader := `{"toast":"Saved."}`
	if got := rec.Header().Get("X-Up-Accept-Layer"); got != wantHeader {
		t.Errorf("X-Up-Accept-Layer = %q, want %q", got, wantHeader)
	}
	if msg, ok := flashToast(t, rec); ok {
		t.Errorf("flash toast = %q, want the cookie cleared after it fires once", msg)
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
