package hybridanalysis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestClient(h http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(h)
	c := NewClient(Config{APIKey: "test-key"})
	c.baseURL = srv.URL
	return c, srv
}

func TestConfigured(t *testing.T) {
	assert.False(t, NewClient(Config{}).Configured())
	assert.True(t, NewClient(Config{APIKey: "x"}).Configured())
}

func TestLookupRequest(t *testing.T) {
	var gotMethod, gotPath, gotKey, gotUA, gotAccept, gotQuery string
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotKey = r.Header.Get("api-key")
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		gotQuery = r.URL.Query().Get("hash")
		w.Write([]byte(`{"sha256s":[],"reports":[]}`))
	})
	defer srv.Close()

	_, err := c.Lookup(context.Background(), "deadbeef")
	assert.Nil(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/search/hash", gotPath)
	assert.Equal(t, "test-key", gotKey)
	assert.Equal(t, "Falcon Sandbox", gotUA)
	assert.Equal(t, "application/json", gotAccept)
	assert.Equal(t, "deadbeef", gotQuery)
}

func TestLookupVerdict(t *testing.T) {
	cases := []struct {
		name      string
		haVerdict string
		want      string
	}{
		{"malicious", "malicious", "malicious"},
		{"suspicious", "suspicious", "suspicious"},
		{"no specific threat → clean", "no specific threat", "clean"},
		{"whitelisted → clean", "whitelisted", "clean"},
		{"unknown string → unknown", "other", "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"reports":[{"verdict":"` + tc.haVerdict + `"}]}`
			c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(body))
			})
			defer srv.Close()

			res, err := c.Lookup(context.Background(), "abc123")
			assert.Nil(t, err)
			assert.Equal(t, tc.want, res.Verdict)
		})
	}
}

func TestLookupEmpty(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"sha256s":["abc123"],"reports":[{"state":"ERROR","verdict":null}]}`))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "abc123")
	assert.Nil(t, err)
	assert.Equal(t, "unknown", res.Verdict)
	assert.Empty(t, res.URL)
}

func TestLookupNotFound(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Requested hash not found"}`))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "abc123")
	assert.Nil(t, err) // 404 is not an error: hash simply unknown
	assert.Equal(t, "unknown", res.Verdict)
	assert.Empty(t, res.URL) // link omitted when there is no record
}

func TestLookupReport(t *testing.T) {
	body := `{"sha256s":["abc123"],"reports":[
		{"state":"SUCCESS","verdict":"suspicious","environment_description":"Linux"},
		{"state":"ERROR","verdict":null,"environment_description":"Android"},
		{"state":"SUCCESS","verdict":"malicious","environment_description":"Windows 11 64 bit"},
		{"state":"SUCCESS","verdict":"no specific threat","environment_description":"Windows 7"}
	]}`
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "abc123")
	assert.Nil(t, err)
	assert.Equal(t, "malicious", res.Verdict)
	assert.Contains(t, res.Summary, "Windows 11 64 bit")
}

func TestLookupURL(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"reports":[{"verdict":"malicious"}]}`))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "deadbeef")
	assert.Nil(t, err)
	assert.Equal(t, "https://www.hybrid-analysis.com/search?query=deadbeef", res.URL)
}

func TestLookupError(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := c.Lookup(context.Background(), "abc123")
	assert.Error(t, err)
}
