package timesketch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fake emulates the Timesketch auth flow: CSRF meta tag on the index page,
// a 200 login page for bad credentials, session cookies, and /users/me/ as
// the authenticated probe.
type fake struct {
	srv *httptest.Server
	mux *http.ServeMux

	mu       sync.Mutex
	user     string
	pass     string
	logins   int
	lastCSRF string
	session  string // currently valid session cookie value, "" = none
}

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{user: "tom", pass: "secret", mux: http.NewServeMux()}

	f.mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><meta name="csrf-token" content="csrf-fixture"></head><body></body></html>`)
	})

	f.mux.HandleFunc("POST /login/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.logins++
		f.lastCSRF = r.FormValue("csrf_token")
		if r.FormValue("username") == f.user && r.FormValue("password") == f.pass {
			f.session = fmt.Sprintf("sess-%d", f.logins)
			http.SetCookie(w, &http.Cookie{Name: "session", Value: f.session, Path: "/"})
		}
		// Timesketch answers both outcomes with a 200 page
		fmt.Fprint(w, "<html>login</html>")
	})

	f.mux.HandleFunc("GET /api/v1/users/me/", func(w http.ResponseWriter, r *http.Request) {
		if f.authed(r) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"meta":{},"objects":[{"username":"tom"}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html>login</html>")
	})

	f.srv = httptest.NewServer(f.mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) authed(r *http.Request) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	ck, err := r.Cookie("session")
	return err == nil && f.session != "" && ck.Value == f.session
}

func (f *fake) expireSession() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.session = ""
}

func (f *fake) loginCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logins
}

// api registers a handler that requires a valid session and answers JSON.
func (f *fake) api(pattern string, h http.HandlerFunc) {
	f.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if !f.authed(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		h(w, r)
	})
}

func (f *fake) client() *Client {
	return NewClient(Config{URL: f.srv.URL, Username: f.user, Password: f.pass})
}

func TestLoginScrapesCSRF(t *testing.T) {
	f := newFake(t)
	f.api("GET /api/v1/sketches/{$}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"meta":{},"objects":[{"id":1,"name":"Case 1"}]}`)
	})

	sketches, err := f.client().ListSketches(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(sketches) != 1 || sketches[0].Name != "Case 1" {
		t.Errorf("unexpected sketches: %+v", sketches)
	}
	if f.lastCSRF != "csrf-fixture" {
		t.Errorf("login posted csrf token %q, want %q", f.lastCSRF, "csrf-fixture")
	}
	if f.loginCount() != 1 {
		t.Errorf("expected 1 login, got %d", f.loginCount())
	}
}

func TestLoginFailure(t *testing.T) {
	f := newFake(t)
	c := NewClient(Config{URL: f.srv.URL, Username: "tom", Password: "wrong"})

	_, err := c.ListSketches(t.Context())
	if err == nil || !strings.Contains(err.Error(), "login failed") {
		t.Errorf("expected login failure, got %v", err)
	}
}

func TestSessionExpiryRetriesOnce(t *testing.T) {
	f := newFake(t)
	f.api("GET /api/v1/sketches/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"meta":{},"objects":[{"id":7,"name":"Case 7"}]}`)
	})

	c := f.client()
	if _, err := c.GetSketch(t.Context(), 7); err != nil {
		t.Fatal(err)
	}

	f.expireSession()
	sketch, err := c.GetSketch(t.Context(), 7)
	if err != nil {
		t.Fatalf("expected re-auth retry to succeed, got %v", err)
	}
	if sketch.ID != 7 {
		t.Errorf("unexpected sketch: %+v", sketch)
	}
	if f.loginCount() != 2 {
		t.Errorf("expected 2 logins, got %d", f.loginCount())
	}
}

func TestPersistentAuthFailureReturnsError(t *testing.T) {
	f := newFake(t)
	f.mux.HandleFunc("GET /api/v1/sketches/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // even with a fresh session
	})

	_, err := f.client().GetSketch(t.Context(), 7)
	if err == nil {
		t.Fatal("expected error")
	}
	if f.loginCount() != 2 {
		t.Errorf("expected exactly one re-login (2 logins), got %d", f.loginCount())
	}
}

func TestGetSketchCopiesMeta(t *testing.T) {
	f := newFake(t)
	f.api("GET /api/v1/sketches/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"meta": {
				"mappings": [{"field": "message", "type": "text"}],
				"attributes": {"intelligence": {"ontology": "intelligence", "value": {"data": [
					{"type": "ipv4", "ioc": "10.0.0.1", "tags": ["bad"], "externalURI": ""}
				]}}}
			},
			"objects": [{"id": 1, "name": "Case 1", "timelines": [{"id": 11, "name": "tl"}]}]
		}`)
	})

	sketch, err := f.client().GetSketch(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sketch.Mappings) != 1 || sketch.Mappings[0].Field != "message" {
		t.Errorf("mappings not copied from meta: %+v", sketch.Mappings)
	}
	data := sketch.Attributes["intelligence"].Values.Data
	if len(data) != 1 || data[0].IOC != "10.0.0.1" {
		t.Errorf("attributes not copied from meta: %+v", sketch.Attributes)
	}
}

func TestGetSketchNotFound(t *testing.T) {
	f := newFake(t)
	f.api("GET /api/v1/sketches/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := f.client().GetSketch(t.Context(), 42)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestExploreAllPaginates(t *testing.T) {
	f := newFake(t)
	froms := []int{}
	f.api("POST /api/v1/sketches/{id}/explore/", func(w http.ResponseWriter, r *http.Request) {
		req := struct {
			Filter Filter `json:"filter"`
		}{}
		json.NewDecoder(r.Body).Decode(&req)
		froms = append(froms, req.Filter.From)

		if req.Filter.From == 0 {
			fmt.Fprint(w, `{"meta": {"has_next": true}, "objects": [
				{"_id": "a", "_source": {"datetime": "2024-01-01T10:00:00Z", "message": "one", "timestamp_desc": "Event Time"}},
				{"_id": "b", "_source": {"datetime": "2024-01-01T11:00:00Z", "message": "two", "timestamp_desc": "Event Time"}}
			]}`)
			return
		}
		fmt.Fprint(w, `{"meta": {"has_next": false}, "objects": [
			{"_id": "c", "_source": {"datetime": "2024-01-01T12:00:00Z", "message": "three", "timestamp_desc": "Event Time"}}
		]}`)
	})

	events, err := f.client().ExploreAll(t.Context(), 1, "*", Filter{Size: 2, Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Message != "one" || events[0].TimestampDesc != "Event Time" || events[0].Datetime.IsZero() {
		t.Errorf("event source not mapped: %+v", events[0])
	}
	if len(froms) != 2 || froms[0] != 0 || froms[1] != 2 {
		t.Errorf("expected from offsets [0 2], got %v", froms)
	}
}

func TestUploadChunks(t *testing.T) {
	old := uploadChunkSize
	uploadChunkSize = 5
	t.Cleanup(func() { uploadChunkSize = old })

	content := []byte("0123456789AB") // 12 bytes -> chunks of 5, 5, 2
	path := filepath.Join(t.TempDir(), "dummy.jsonl")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	type chunk struct {
		fields map[string]string
		data   []byte
	}
	chunks := []chunk{}

	f := newFake(t)
	f.mux.HandleFunc("POST /api/v1/upload/", func(w http.ResponseWriter, r *http.Request) {
		if !f.authed(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fields := map[string]string{}
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
		fh, _, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		buf := &bytes.Buffer{}
		buf.ReadFrom(fh)
		chunks = append(chunks, chunk{fields: fields, data: buf.Bytes()})
		w.WriteHeader(http.StatusCreated)
	})

	if err := f.client().Upload(t.Context(), 3, path); err != nil {
		t.Fatal(err)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	reassembled := []byte{}
	for i, c := range chunks {
		if got := c.fields["chunk_index"]; got != strconv.Itoa(i) {
			t.Errorf("chunk %d: chunk_index = %s", i, got)
		}
		if got := c.fields["chunk_byte_offset"]; got != strconv.Itoa(i*5) {
			t.Errorf("chunk %d: chunk_byte_offset = %s", i, got)
		}
		if got := c.fields["chunk_total_chunks"]; got != "3" {
			t.Errorf("chunk %d: chunk_total_chunks = %s", i, got)
		}
		if got := c.fields["total_file_size"]; got != "12" {
			t.Errorf("chunk %d: total_file_size = %s", i, got)
		}
		if got := c.fields["name"]; got != "dummy.jsonl" {
			t.Errorf("chunk %d: name = %s", i, got)
		}
		if got := c.fields["sketch_id"]; got != "3" {
			t.Errorf("chunk %d: sketch_id = %s", i, got)
		}
		if got := c.fields["provider"]; got != "Dagobert" {
			t.Errorf("chunk %d: provider = %s", i, got)
		}
		reassembled = append(reassembled, c.data...)
	}
	if !bytes.Equal(reassembled, content) {
		t.Errorf("reassembled %q, want %q", reassembled, content)
	}
}

func TestErrorIncludesBody(t *testing.T) {
	f := newFake(t)
	f.mux.HandleFunc("GET /api/v1/sketches/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	})

	_, err := f.client().ListSketches(t.Context())
	if err == nil || !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected error with status and body, got %v", err)
	}
}

func TestUnconfiguredClient(t *testing.T) {
	c := NewClient(Config{})
	if c.Configured() {
		t.Error("client with empty URL must not report configured")
	}

	if _, err := c.ListSketches(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("ListSketches: expected ErrNotConfigured, got %v", err)
	}
	if err := c.Upload(context.Background(), 1, "nope"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Upload: expected ErrNotConfigured, got %v", err)
	}
	if err := c.Login(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Login: expected ErrNotConfigured, got %v", err)
	}
}
