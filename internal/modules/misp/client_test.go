package misp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestClient(url string, h http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(h)
	return NewClient(Config{URL: srv.URL + url, APIKey: "test-key"}), srv
}

func respond(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }
}

func TestConfigured(t *testing.T) {
	assert.False(t, NewClient(Config{}).Configured())
	assert.False(t, NewClient(Config{URL: "https://misp"}).Configured())
	assert.False(t, NewClient(Config{APIKey: "x"}).Configured())
	assert.True(t, NewClient(Config{URL: "https://misp", APIKey: "x"}).Configured())
}

func TestLookupRequest(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotBody string
	c, srv := newTestClient("", func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{"response":{"Attribute":[]}}`))
	})
	defer srv.Close()

	_, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Nil(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/attributes/restSearch", gotPath)
	assert.Equal(t, "test-key", gotAuth)
	assert.Contains(t, gotBody, `"value":"1.2.3.4"`)
}

func TestLookupNoMatch(t *testing.T) {
	c, srv := newTestClient("", respond(`{"response":{"Attribute":[]}}`))
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Nil(t, err)
	assert.Equal(t, "unknown", res.Verdict)
	assert.Empty(t, res.URL)
	assert.Contains(t, res.Summary, "No record found in MISP")
}

func TestLookupVerdict(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		verdict string
	}{
		{"to_ids high → malicious", `{"response":{"Attribute":[{"event_id":"1","to_ids":true,"Event":{"threat_level_id":"1"}}]}}`, "malicious"},
		{"to_ids medium → malicious", `{"response":{"Attribute":[{"event_id":"1","to_ids":true,"Event":{"threat_level_id":"2"}}]}}`, "malicious"},
		{"to_ids low → suspicious", `{"response":{"Attribute":[{"event_id":"1","to_ids":true,"Event":{"threat_level_id":"3"}}]}}`, "suspicious"},
		{"to_ids undefined → suspicious", `{"response":{"Attribute":[{"event_id":"1","to_ids":true,"Event":{"threat_level_id":"4"}}]}}`, "suspicious"},
		{"no to_ids → clean", `{"response":{"Attribute":[{"event_id":"1","to_ids":false,"Event":{"threat_level_id":"1"}}]}}`, "clean"},
		{"no match → unknown", `{"response":{"Attribute":[]}}`, "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := newTestClient("", respond(tc.body))
			defer srv.Close()

			res, err := c.Lookup(context.Background(), "1.2.3.4")
			assert.Nil(t, err)
			assert.Equal(t, tc.verdict, res.Verdict)
		})
	}
}

// TestLookupSummary covers counts, tag capping/sorting, and the deciding event's link.
func TestLookupSummary(t *testing.T) {
	body := `{"response":{"Attribute":[
		{"event_id":"10","to_ids":false,"Event":{"threat_level_id":"1"},"Tag":[{"name":"tlp:green"},{"name":"zeta"}]},
		{"event_id":"20","to_ids":true,"Event":{"threat_level_id":"3"},"Tag":[{"name":"misp-galaxy:threat-actor=\"APT28\""},{"name":"beta"}]},
		{"event_id":"30","to_ids":true,"Event":{"threat_level_id":"2"},"Tag":[{"name":"alpha"},{"name":"misp-galaxy:malware=\"X-Agent\""}]},
		{"event_id":"30","to_ids":true,"Event":{"threat_level_id":"2"},"Tag":[{"name":"alpha"}]}
	]}}`
	c, srv := newTestClient("/", respond(body))
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Nil(t, err)
	assert.Equal(t, "malicious", res.Verdict)
	assert.Contains(t, res.Summary, "Matches: 4 attributes in 3 events")
	assert.Contains(t, res.Summary, "Threat level: High")
	assert.Contains(t, res.Summary, `Tags: alpha, beta, misp-galaxy:malware="X-Agent", misp-galaxy:threat-actor="APT28", tlp:green`+"\n")
	assert.Equal(t, srv.URL+"/events/view/30", res.URL)
}

func TestLookupError(t *testing.T) {
	c, srv := newTestClient("", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()
	_, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Error(t, err)

	c, srv2 := newTestClient("", respond(`{not json`))
	defer srv2.Close()
	_, err = c.Lookup(context.Background(), "1.2.3.4")
	assert.Error(t, err)
}

func TestVerifyAuth(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		c, srv := newTestClient("", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		})
		err := c.Verify(context.Background())
		srv.Close()
		assert.ErrorContains(t, err, "authentication failed")
	}
}
