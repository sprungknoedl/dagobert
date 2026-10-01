package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/auth"
	"github.com/sprungknoedl/dagobert/internal/model"
)

func postTaskSave(h *Handler, cid, owner string) *httptest.ResponseRecorder {
	form := url.Values{
		"Type":    {"Analysis"},
		"Task":    {"Review logs"},
		"OwnerID": {owner},
	}
	r := httptest.NewRequest(http.MethodPost, "/cases/"+cid+"/tasks/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("cid", cid)
	r.SetPathValue("id", "new")
	rec := httptest.NewRecorder()
	h.TaskSave(rec, r)
	return rec
}

func TestTaskSaveRejectsInvalidOwner(t *testing.T) {
	db := setupArchiveDB(t)
	kase := model.Case{ID: "case01", Name: "Operation Test"}
	if err := db.SaveCase(kase); err != nil {
		t.Fatal(err)
	}
	h := &Handler{Store: db, ACL: auth.NewACL(db)}

	t.Run("unknown user id", func(t *testing.T) {
		rec := postTaskSave(h, kase.ID, "does-not-exist")
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got status %d, want 422; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("builtin user id", func(t *testing.T) {
		rec := postTaskSave(h, kase.ID, model.SystemUser.ID)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("got status %d, want 422; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("real non-builtin user id is accepted", func(t *testing.T) {
		user := model.User{ID: "u1", Name: "Analyst", Login: "analyst"}
		if err := db.SaveUser(user); err != nil {
			t.Fatal(err)
		}

		rec := postTaskSave(h, kase.ID, user.ID)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want 303; body: %s", rec.Code, rec.Body.String())
		}

		list, err := db.ListTasks(kase.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].OwnerID == nil || *list[0].OwnerID != user.ID {
			t.Errorf("got tasks %+v, want one task owned by %s", list, user.ID)
		}
	})

	t.Run("empty owner stores unassigned", func(t *testing.T) {
		rec := postTaskSave(h, kase.ID, "")
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want 303; body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestTaskImportResolvesOwnerByLogin(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	if err := db.SaveUser(model.User{ID: "u1", Name: "Analyst", Login: "analyst"}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db}
	body := "ID,Type,Task,Done,Owner,Due Date,Custom\n" +
		"t1,Analysis,Known owner,false,analyst,2024-01-01T00:00:00Z,\n" +
		"t2,Analysis,Unknown owner,false,nobody,2024-01-01T00:00:00Z,\n" +
		"t3,Analysis,No owner,false,,2024-01-01T00:00:00Z,\n"
	r := newGenericCSVImportRequest(t, "/cases/{cid}/tasks/import/csv", kase.ID, body)
	rec := httptest.NewRecorder()
	h.TaskImport(rec, r)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body: %s", rec.Code, rec.Body.String())
	}

	list, err := db.ListTasks(kase.ID)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]model.Task{}
	for _, task := range list {
		byID[task.ID] = task
	}

	if got := byID["t1"].OwnerID; got == nil || *got != "u1" {
		t.Errorf("t1 owner = %v, want u1", got)
	}
	if got := byID["t2"].OwnerID; got != nil {
		t.Errorf("t2 owner = %v, want unassigned (unknown login must not fail the row)", *got)
	}
	if got := byID["t3"].OwnerID; got != nil {
		t.Errorf("t3 owner = %v, want unassigned (empty login must not fail the row)", *got)
	}
}

func TestTaskListJSONIncludesOwner(t *testing.T) {
	db := setupArchiveDB(t)
	kase := model.Case{ID: "case01", Name: "Operation Test"}
	if err := db.SaveCase(kase); err != nil {
		t.Fatal(err)
	}
	owner := model.User{ID: "u1", Name: "Analyst", Login: "analyst"}
	if err := db.SaveUser(owner); err != nil {
		t.Fatal(err)
	}
	task := model.Task{ID: "t1", CaseID: kase.ID, Type: "Analysis", Task: "Review", OwnerID: &owner.ID}
	if err := db.SaveTask(kase.ID, task); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db}
	r := httptest.NewRequest(http.MethodGet, "/cases/"+kase.ID+"/tasks/", nil)
	r.Header.Set("Accept", "application/json")
	r.SetPathValue("cid", kase.ID)
	rec := httptest.NewRecorder()
	h.TaskList(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var got []model.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d tasks, want 1", len(got))
	}
	if got[0].Owner.ID != owner.ID || got[0].Owner.Name != owner.Name {
		t.Errorf("got owner %+v, want ID=%s Name=%s", got[0].Owner, owner.ID, owner.Name)
	}
}

func TestTaskListRendersOwner(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	owner := model.User{ID: "u1", Name: "Analyst Jane", Login: "analyst"}
	if err := db.SaveUser(owner); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveTask(kase.ID, model.Task{ID: "t1", CaseID: kase.ID, Type: "Analysis", Task: "Owned task", OwnerID: &owner.ID}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveTask(kase.ID, model.Task{ID: "t2", CaseID: kase.ID, Type: "Analysis", Task: "Unassigned task"}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db, ACL: auth.NewACL(db)}
	r := httptest.NewRequest(http.MethodGet, "/cases/"+kase.ID+"/tasks/", nil)
	r.SetPathValue("cid", kase.ID)
	rec := httptest.NewRecorder()
	h.TaskList(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, owner.Name) {
		t.Errorf("response missing owner name %q", owner.Name)
	}
}

func TestTaskEditOwnerOptions(t *testing.T) {
	db := setupArchiveDB(t)
	kase := seedCase(t, db)
	if err := db.SaveUser(model.User{ID: "u1", Name: "Analyst Jane", Login: "analyst"}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: db, ACL: auth.NewACL(db)}
	r := httptest.NewRequest(http.MethodGet, "/cases/"+kase.ID+"/tasks/new", nil)
	r.SetPathValue("cid", kase.ID)
	r.SetPathValue("id", "new")
	rec := httptest.NewRecorder()
	h.TaskEdit(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Unassigned") {
		t.Error("owner select missing the Unassigned option")
	}
	if !strings.Contains(body, "Analyst Jane") {
		t.Error("owner select missing the assignable user")
	}
	for _, builtin := range []model.User{model.SystemUser, model.DonaldUser, model.McpUser} {
		if strings.Contains(body, builtin.Name) {
			t.Errorf("owner select must not offer builtin user %q", builtin.Name)
		}
	}
}
