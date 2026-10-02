package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/auth"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
)

// TestCaseDelete checks that deleting a case removes its files on disk, the
// FK-less enrichment and evidence-log rows, and the case's ACL policies, both
// in the database and in the loaded enforcer.
func TestCaseDelete(t *testing.T) {
	db := setupArchiveDB(t)
	seedCase(t, db)
	t.Chdir(t.TempDir())

	acl := auth.NewACL(db)
	if err := db.SaveUser(model.User{ID: "user01", Name: "Alice", Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err := acl.SaveCasePermissions("case01", []string{"user01"}); err != nil {
		t.Fatal(err)
	}
	if err := db.SetEnrichment(model.Enrichment{CaseID: "case01", ObjectType: "Indicator", ObjectID: "ind01", Module: "VirusTotal"}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveEvidenceLog("case01", model.EvidenceLog{EvidenceID: "evid01", Event: model.EvidenceLogUploaded}); err != nil {
		t.Fatal(err)
	}

	evidenceDir := filepath.Join(model.DataDir, "evidences", "case01")
	malwareDir := filepath.Join(model.DataDir, "malware", "case01")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(malwareDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "dummy.txt"), []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(malwareDir, "dummy.zip"), []byte("malware"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db, ACL: acl}
	r := httptest.NewRequest(http.MethodDelete, "/cases/case01?confirm=yes", nil)
	r.SetPathValue("cid", "case01")
	rec := httptest.NewRecorder()

	h.CaseDelete(rec, r)

	if _, err := os.Stat(evidenceDir); !os.IsNotExist(err) {
		t.Errorf("evidence dir still exists: %v", err)
	}
	if _, err := os.Stat(malwareDir); !os.IsNotExist(err) {
		t.Errorf("malware dir still exists: %v", err)
	}
	if _, err := db.GetCase("case01"); err == nil {
		t.Errorf("case still exists after delete")
	}
	if enr, _ := db.ListEnrichments("Indicator", "ind01"); len(enr) != 0 {
		t.Errorf("enrichments left behind: %v", enr)
	}
	if logs, _ := db.ListEvidenceLogs("case01"); len(logs) != 0 {
		t.Errorf("evidence logs left behind: %v", logs)
	}
	if users, _ := db.GetCasePermissions("case01"); len(users) != 0 {
		t.Errorf("policies left behind: %v", users)
	}
	if acl.Allowed("user01", "/cases/case01/summary/", http.MethodGet) {
		t.Errorf("enforcer still grants access to the deleted case")
	}
}

func postCaseSave(h *Handler, kase model.Case, assignee string) *httptest.ResponseRecorder {
	form := url.Values{
		"Name":      {kase.Name},
		"Severity":  {kase.Severity},
		"Assignees": {assignee},
	}
	r := httptest.NewRequest(http.MethodPost, "/cases/"+kase.ID+"/edit", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("cid", kase.ID)
	rec := httptest.NewRecorder()
	h.CaseSave(rec, r)
	return rec
}

func TestCaseSaveRejectsInvalidAssignees(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	h := &Handler{Store: db, ACL: auth.NewACL(db), Timesketch: timesketch.NewClient(timesketch.Config{})}

	t.Run("unknown user id", func(t *testing.T) {
		rec := postCaseSave(h, kase, "does-not-exist")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got status %d, want 422", rec.Code)
		}

		got, err := db.GetCaseWithAssignees(kase.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Assignees) != 0 {
			t.Errorf("assignees were saved despite validation failure: %v", got.Assignees)
		}
	})

	t.Run("builtin user id", func(t *testing.T) {
		rec := postCaseSave(h, kase, model.SystemUser.ID)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got status %d, want 422", rec.Code)
		}

		got, err := db.GetCaseWithAssignees(kase.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Assignees) != 0 {
			t.Errorf("assignees were saved despite validation failure: %v", got.Assignees)
		}
	})

	t.Run("real non-builtin user id is accepted", func(t *testing.T) {
		user := model.User{ID: "u1", Name: "Analyst", Login: "analyst"}
		if err := db.SaveUser(user); err != nil {
			t.Fatal(err)
		}

		rec := postCaseSave(h, kase, user.ID)
		if rec.Code != http.StatusSeeOther && rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want a redirect", rec.Code)
		}

		got, err := db.GetCaseWithAssignees(kase.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Assignees) != 1 || got.Assignees[0].ID != user.ID {
			t.Errorf("got assignees %v, want [%s]", got.Assignees, user.ID)
		}
	})
}

// TestCaseSaveIgnoresBodyID checks that an ID in the body can't redirect the
// save to a different case than the one in the (ACL-checked) path.
func TestCaseSaveIgnoresBodyID(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	other := model.Case{ID: "case02", Name: "Other", Severity: "Low"}
	if err := db.SaveCase(other); err != nil {
		t.Fatal(err)
	}
	h := &Handler{Store: db, ACL: auth.NewACL(db), Timesketch: timesketch.NewClient(timesketch.Config{})}

	form := url.Values{"ID": {other.ID}, "Name": {"Renamed"}, "Severity": {"High"}}
	r := httptest.NewRequest(http.MethodPost, "/cases/"+kase.ID+"/edit", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("cid", kase.ID)
	h.CaseSave(httptest.NewRecorder(), r)

	got, err := db.GetCase(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != other.Name {
		t.Errorf("case %s was renamed to %q via the body ID", other.ID, got.Name)
	}
	if got, _ := db.GetCase(kase.ID); got.Name != "Renamed" {
		t.Errorf("got name %q for %s, want Renamed", got.Name, kase.ID)
	}
}

// TestCaseSaveJSON creates a case through the JSON API with assignees as user
// ids and custom attributes, and checks that both are stored.
func TestCaseSaveJSON(t *testing.T) {
	db := setupArchiveDB(t)
	user := model.User{ID: "u1", Name: "Analyst", Login: "analyst"}
	if err := db.SaveUser(user); err != nil {
		t.Fatal(err)
	}
	h := &Handler{Store: db, ACL: auth.NewACL(db), Timesketch: timesketch.NewClient(timesketch.Config{})}

	r := newJSONRequest(t, "/cases/new", map[string]any{
		"Name":      "Operation JSON",
		"Severity":  "High",
		"Assignees": []string{user.ID},
		"Custom":    map[string]string{"Ticket": "INC-1"},
	})
	r.SetPathValue("cid", "new")
	rec := httptest.NewRecorder()
	h.CaseSave(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var resp model.Case
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetCaseWithAssignees(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Operation JSON" || got.Severity != "High" {
		t.Errorf("got name %q severity %q, want Operation JSON/High", got.Name, got.Severity)
	}
	if len(got.Assignees) != 1 || got.Assignees[0].ID != user.ID {
		t.Errorf("got assignees %v, want [%s]", got.Assignees, user.ID)
	}
	if got.Custom["Ticket"] != "INC-1" {
		t.Errorf("got custom %v, want Ticket=INC-1", got.Custom)
	}
}
