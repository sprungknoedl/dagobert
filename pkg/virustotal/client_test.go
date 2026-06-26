package virustotal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestClient wires a client at an httptest server with handler h.
func newTestClient(h http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(h)
	c := NewClient(Config{APIKey: "secret-key"})
	c.baseURL = srv.URL
	return c, srv
}

func TestConfigured(t *testing.T) {
	assert.False(t, NewClient(Config{}).Configured())
	assert.True(t, NewClient(Config{APIKey: "x"}).Configured())
}

func TestLookupRequestShaping(t *testing.T) {
	cases := []struct {
		typ  string
		val  string
		path string
	}{
		{"Hash", "abc123", "/files/abc123"},
		{"IP", "185.220.101.5", "/ip_addresses/185.220.101.5"},
		{"Domain", "evil.example", "/domains/evil.example"},
		// VT URL identifier: base64url(value), '=' padding stripped
		{"URL", "http://evil.example/x", "/urls/aHR0cDovL2V2aWwuZXhhbXBsZS94"},
	}

	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			var gotPath, gotKey, gotAccept string
			c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotKey = r.Header.Get("x-apikey")
				gotAccept = r.Header.Get("Accept")
				w.Write([]byte(`{"data":{"attributes":{"last_analysis_stats":{"harmless":70}}}}`))
			})
			defer srv.Close()

			_, err := c.Lookup(context.Background(), tc.typ, tc.val)
			assert.Nil(t, err)
			assert.Equal(t, tc.path, gotPath)
			assert.Equal(t, "secret-key", gotKey)
			assert.Equal(t, "application/json", gotAccept)
		})
	}
}

func TestLookupVerdictDerivation(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		verdict string
		score   string
	}{
		{"malicious", `{"data":{"attributes":{"last_analysis_stats":{"malicious":3,"suspicious":1,"harmless":60,"undetected":10}}}}`, "malicious", "3/74"},
		{"suspicious", `{"data":{"attributes":{"last_analysis_stats":{"malicious":0,"suspicious":2,"harmless":60,"undetected":10}}}}`, "suspicious", "0/72"},
		{"clean", `{"data":{"attributes":{"last_analysis_stats":{"malicious":0,"suspicious":0,"harmless":60,"undetected":10}}}}`, "clean", "0/70"},
		{"unknown", `{"data":{"attributes":{"last_analysis_stats":{"malicious":0,"suspicious":0,"harmless":0,"undetected":0}}}}`, "unknown", "0/0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(tc.body))
			})
			defer srv.Close()

			res, err := c.Lookup(context.Background(), "IP", "1.2.3.4")
			assert.Nil(t, err)
			assert.Equal(t, tc.verdict, res.Verdict)
			assert.Equal(t, tc.score, res.Score)
			assert.Equal(t, "https://www.virustotal.com/gui/search/1.2.3.4", res.URL)
			assert.Contains(t, res.Summary, "Verdict: "+tc.verdict)
		})
	}
}

func TestLookupFoldsDetections(t *testing.T) {
	body := `{"data":{"attributes":{
		"last_analysis_stats":{"malicious":2,"harmless":50},
		"last_analysis_results":{
			"EngineA":{"category":"malicious","result":"Trojan.X"},
			"EngineB":{"category":"malicious","result":"Trojan.X"},
			"EngineC":{"category":"suspicious","result":"Heur.Y"},
			"EngineD":{"category":"harmless","result":""}
		}}}}`
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "Hash", "deadbeef")
	assert.Nil(t, err)
	// deduplicated and sorted
	assert.Contains(t, res.Summary, "Top detections: Heur.Y, Trojan.X")
}

func TestLookupNotFound(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"NotFoundError"}}`))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "Domain", "unknown.example")
	assert.Nil(t, err)
	assert.Equal(t, "unknown", res.Verdict) // 404 conflated into "unknown"
	assert.Empty(t, res.URL)                // link omitted when there is no record
}

func TestLookupServerError(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := c.Lookup(context.Background(), "IP", "1.2.3.4")
	assert.Error(t, err)
}

func TestVerify(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		var gotPath, gotKey string
		c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotKey = r.Header.Get("x-apikey")
			w.Write([]byte(`{"data":{}}`))
		})
		defer srv.Close()

		assert.Nil(t, c.Verify(context.Background()))
		assert.Equal(t, "/users/secret-key", gotPath)
		assert.Equal(t, "secret-key", gotKey)
	})

	t.Run("bad key", func(t *testing.T) {
		c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		defer srv.Close()

		assert.Error(t, c.Verify(context.Background()))
	})
}
