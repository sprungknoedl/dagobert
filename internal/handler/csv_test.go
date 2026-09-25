package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
)

func newGenericCSVImportRequest(t *testing.T, path, cid, body string) *http.Request {
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

	r := httptest.NewRequest(http.MethodPost, path, &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if cid != "" {
		r.SetPathValue("cid", cid)
	}
	return r
}

// TestImportCSVSample checks every type's ?sample download imports cleanly.
func TestImportCSVSample(t *testing.T) {
	tests := []struct {
		name     string
		needsCID bool
		path     string
		do       func(h *Handler, w http.ResponseWriter, r *http.Request)
	}{
		{"cases", false, "/cases/import/csv", (*Handler).CaseImport},
		{"assets", true, "/cases/{cid}/assets/import/csv", (*Handler).AssetImport},
		{"events", true, "/cases/{cid}/events/import/csv", (*Handler).EventImportCSV},
		{"evidences", true, "/cases/{cid}/evidences/import/csv", (*Handler).EvidenceImport},
		{"indicators", true, "/cases/{cid}/indicators/import/csv", (*Handler).IndicatorImportCSV},
		{"malware", true, "/cases/{cid}/malware/import/csv", (*Handler).MalwareImport},
		{"notes", true, "/cases/{cid}/notes/import/csv", (*Handler).NoteImport},
		{"tasks", true, "/cases/{cid}/tasks/import/csv", (*Handler).TaskImport},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := setupArchiveDB(t)
			cid := ""
			if tc.needsCID {
				cid = "case01"
				if err := db.SaveCase(model.Case{ID: cid, Name: "Sample import case"}); err != nil {
					t.Fatal(err)
				}
			}

			h := &Handler{Store: db}
			sr := httptest.NewRequest(http.MethodGet, tc.path+"?sample", nil)
			sr.SetPathValue("cid", cid)
			srec := httptest.NewRecorder()
			tc.do(h, srec, sr)
			if srec.Code != http.StatusOK {
				t.Fatalf("sample status = %d, want %d", srec.Code, http.StatusOK)
			}

			r := newGenericCSVImportRequest(t, tc.path, cid, srec.Body.String())
			rec := httptest.NewRecorder()
			tc.do(h, rec, r)

			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
			}
		})
	}
}

// TestImportCSVRoundtrip exports one record per type (with a comma in Custom
// to exercise quoting) and re-imports it, which must update, not duplicate.
func TestImportCSVRoundtrip(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T, db *model.Store, cid string)
		export  func(h *Handler, w http.ResponseWriter, r *http.Request)
		importF func(h *Handler, w http.ResponseWriter, r *http.Request)
		count   func(t *testing.T, db *model.Store, cid string) int
	}{
		{
			name: "assets",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				obj := model.Asset{
					ID: "a1", CaseID: cid, Status: "Compromised", Type: "Server", Name: "WKS-77",
					Addr: "10.0.0.5", Notes: "observed beaconing",
					Custom: model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveAsset(cid, obj); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).AssetExport,
			importF: (*Handler).AssetImport,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListAssets(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
		{
			name: "events",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				asset := model.Asset{ID: "a1", CaseID: cid, Status: "Compromised", Type: "Server", Name: "WKS-77"}
				if err := db.SaveAsset(cid, asset); err != nil {
					t.Fatal(err)
				}
				ind := model.Indicator{ID: "i1", CaseID: cid, Status: "Confirmed", Type: "IP", Value: "198.51.100.9", TLP: "TLP:RED"}
				if err := db.SaveIndicator(cid, ind, false); err != nil {
					t.Fatal(err)
				}
				obj := model.Event{
					ID: "e1", CaseID: cid, Time: model.Time(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)),
					Type: "C2", Assets: []model.Asset{{ID: asset.ID}}, Indicators: []model.Indicator{{ID: ind.ID}},
					Event: "beacon observed", Raw: "raw log line", Source: "edr",
					Custom: model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveEvent(cid, obj, false); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).EventExport,
			importF: (*Handler).EventImportCSV,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListEvents(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
		{
			name: "evidences",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				obj := model.Evidence{
					ID: "ev1", CaseID: cid, Type: "File", Name: "eventlog.evtx", Hash: "deadbeef",
					Size: 1024, Fileless: true, Notes: "chain of custody, verified",
					StartsAt: model.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					EndsAt:   model.Time(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)),
					Custom:   model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveEvidence(cid, obj); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).EvidenceExport,
			importF: (*Handler).EvidenceImport,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListEvidences(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
		{
			name: "indicators",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				obj := model.Indicator{
					ID: "i1", CaseID: cid, Status: "Confirmed", Type: "IP", Value: "198.51.100.9",
					TLP: "TLP:RED", Source: "osint, verified", Notes: "blocked at firewall",
					Custom: model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveIndicator(cid, obj, false); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).IndicatorExportCSV,
			importF: (*Handler).IndicatorImportCSV,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListIndicators(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
		{
			name: "malware",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				asset := model.Asset{ID: "a1", CaseID: cid, Status: "Compromised", Type: "Server", Name: "WKS-77"}
				if err := db.SaveAsset(cid, asset); err != nil {
					t.Fatal(err)
				}
				obj := model.Malware{
					ID: "m1", CaseID: cid, Status: "Malicious", Path: `C:\temp\evil.exe`, Hash: "abc123",
					Asset: model.Asset{ID: asset.ID}, Notes: "packed, obfuscated",
					Custom: model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveMalware(cid, obj); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).MalwareExport,
			importF: (*Handler).MalwareImport,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListMalware(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
		{
			name: "notes",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				obj := model.Note{
					ID: "n1", CaseID: cid, Title: "Lead", Category: "General",
					Description: "found evidence, still analyzing",
					Custom:      model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveNote(cid, obj); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).NoteExport,
			importF: (*Handler).NoteImport,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListNotes(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
		{
			name: "tasks",
			seed: func(t *testing.T, db *model.Store, cid string) {
				t.Helper()
				obj := model.Task{
					ID: "t1", CaseID: cid, Type: "Analysis", Task: "Review logs, escalate if needed",
					Done: true, Owner: "alice", DateDue: model.Time(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					Custom: model.Custom{"context": "seen twice, confirmed"},
				}
				if err := db.SaveTask(cid, obj); err != nil {
					t.Fatal(err)
				}
			},
			export:  (*Handler).TaskExport,
			importF: (*Handler).TaskImport,
			count: func(t *testing.T, db *model.Store, cid string) int {
				t.Helper()
				list, err := db.ListTasks(cid)
				if err != nil {
					t.Fatal(err)
				}
				return len(list)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := setupArchiveDB(t)
			cid := "case01"
			if err := db.SaveCase(model.Case{ID: cid, Name: "Round-trip case"}); err != nil {
				t.Fatal(err)
			}
			tc.seed(t, db, cid)

			h := &Handler{Store: db}

			er := httptest.NewRequest(http.MethodGet, "/export", nil)
			er.SetPathValue("cid", cid)
			erec := httptest.NewRecorder()
			tc.export(h, erec, er)
			if erec.Code != http.StatusOK {
				t.Fatalf("export status = %d, want %d; body: %s", erec.Code, http.StatusOK, erec.Body.String())
			}

			before := tc.count(t, db, cid)

			ir := newGenericCSVImportRequest(t, "/import", cid, erec.Body.String())
			irec := httptest.NewRecorder()
			tc.importF(h, irec, ir)
			if irec.Code != http.StatusSeeOther {
				t.Fatalf("import status = %d, want %d; body: %s", irec.Code, http.StatusSeeOther, irec.Body.String())
			}

			after := tc.count(t, db, cid)
			if after != before {
				t.Errorf("record count changed from %d to %d — a round-trip import by the same ID should update, not duplicate", before, after)
			}
		})
	}
}

// TestCaseImportRoundtrip is TestImportCSVRoundtrip for cases, which have no {cid}.
func TestCaseImportRoundtrip(t *testing.T) {
	db := setupArchiveDB(t)
	kase := model.Case{
		ID: "case01", Name: "Operation Comma", Severity: "High", Classification: "Phishing, targeted",
		Outcome: "True positive", Summary: "multi, comma, summary", OpenedAt: model.Date(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		Custom: model.Custom{"context": "seen twice, confirmed"},
	}
	if err := db.SaveCase(kase); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db}

	er := httptest.NewRequest(http.MethodGet, "/cases/export/csv", nil)
	erec := httptest.NewRecorder()
	h.CaseExport(erec, er)
	if erec.Code != http.StatusOK {
		t.Fatalf("export status = %d, want %d; body: %s", erec.Code, http.StatusOK, erec.Body.String())
	}

	before, err := db.ListCases()
	if err != nil {
		t.Fatal(err)
	}

	ir := newGenericCSVImportRequest(t, "/cases/import/csv", "", erec.Body.String())
	irec := httptest.NewRecorder()
	h.CaseImport(irec, ir)
	if irec.Code != http.StatusSeeOther {
		t.Fatalf("import status = %d, want %d; body: %s", irec.Code, http.StatusSeeOther, irec.Body.String())
	}

	after, err := db.ListCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("case count changed from %d to %d — a round-trip import by the same ID should update, not duplicate", len(before), len(after))
	}
}

var testCSVSpec = views.CSVSpec{
	Columns: []string{"ID", "Name", "Notes"},
	Sample:  []string{"", "Sample", "A fictional sample row."},
}

var csvHeaderDetectionFixtures = struct {
	matchingHeader   string
	noMatchingHeader string
	semicolonDelim   string
	quotedComma      string
	wrongFieldCount  string
}{
	matchingHeader: "id,NAME,notes\n" + // case-insensitive match
		"1,Alice,first\n" +
		"2,Bob,second\n",
	noMatchingHeader: "1,Alice,first\n" +
		"2,Bob,second\n",
	semicolonDelim: "ID;Name;Notes\n" +
		"1;Alice;first\n" +
		"2;Bob;second\n",
	quotedComma: "ID,Name,Notes\n" +
		"1,Alice,\"hello, world\"\n",
	wrongFieldCount: "ID,Name,Notes\n" +
		"1,Alice,first,extra\n" +
		"2,Bob,second\n",
}

func TestImportCSVHeaderMatch(t *testing.T) {
	db := setupArchiveDB(t)
	var got [][]string
	r := newGenericCSVImportRequest(t, "/import", "", csvHeaderDetectionFixtures.matchingHeader)
	rec := httptest.NewRecorder()
	ImportCSV(db, rec, r, "/", testCSVSpec, func(tx *model.Store, row []string) error {
		got = append(got, row)
		return nil
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	want := [][]string{{"1", "Alice", "first"}, {"2", "Bob", "second"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v — the header row must not be treated as data", got, want)
	}
}

func TestImportCSVHeaderMismatch(t *testing.T) {
	db := setupArchiveDB(t)
	var got [][]string
	r := newGenericCSVImportRequest(t, "/import", "", csvHeaderDetectionFixtures.noMatchingHeader)
	rec := httptest.NewRecorder()
	ImportCSV(db, rec, r, "/", testCSVSpec, func(tx *model.Store, row []string) error {
		got = append(got, row)
		return nil
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	want := [][]string{{"1", "Alice", "first"}, {"2", "Bob", "second"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v — a headerless file must not lose its first row", got, want)
	}
}

func TestPreviewCSV(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		partial   bool
		wantRows  [][]string
		wantError string
	}{
		{"matching header", csvHeaderDetectionFixtures.matchingHeader, false, [][]string{{"1", "Alice", "first"}, {"2", "Bob", "second"}}, ""},
		{"semicolon delimited", csvHeaderDetectionFixtures.semicolonDelim, false, [][]string{{"1", "Alice", "first"}, {"2", "Bob", "second"}}, ""},
		{"quoted comma", csvHeaderDetectionFixtures.quotedComma, false, [][]string{{"1", "Alice", "hello, world"}}, ""},
		{"headerless", csvHeaderDetectionFixtures.noMatchingHeader, false, [][]string{{"1", "Alice", "first"}, {"2", "Bob", "second"}}, ""},
		{"wrong field count", csvHeaderDetectionFixtures.wrongFieldCount, false, nil, "Row 2:"},
		{"partial drops cut-off last row", "ID,Name,Notes\n1,Alice,first\n2,Bob,sec", true, [][]string{{"1", "Alice", "first"}}, ""},
		{"partial ignores cut-off row error", "ID,Name,Notes\n1,Alice,first\n2,Bob", true, [][]string{{"1", "Alice", "first"}}, ""},
		{"first 10 rows only", "ID,Name,Notes\n" + strings.Repeat("1,Alice,first\n", 12), false, slices.Repeat([][]string{{"1", "Alice", "first"}}, 10), ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := previewCSV(strings.NewReader(tc.text), testCSVSpec, tc.partial)
			if tc.wantError != "" {
				if !strings.HasPrefix(got.Error, tc.wantError) {
					t.Errorf("error = %q, want prefix %q", got.Error, tc.wantError)
				}
				return
			}
			if got.Error != "" {
				t.Fatalf("error = %q, want none", got.Error)
			}
			if !reflect.DeepEqual(got.Rows, tc.wantRows) {
				t.Errorf("rows = %v, want %v", got.Rows, tc.wantRows)
			}
		})
	}
}
