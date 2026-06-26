package abuseipdb

import (
	"context"
	"fmt"
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

func TestLookupRequestShaping(t *testing.T) {
	var gotPath, gotKey, gotAccept string
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		gotKey = r.Header.Get("Key")
		gotAccept = r.Header.Get("Accept")
		w.Write([]byte(`{"data":{"abuseConfidenceScore":0,"totalReports":0}}`))
	})
	defer srv.Close()

	_, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Nil(t, err)
	assert.Contains(t, gotPath, "/check")
	assert.Contains(t, gotPath, "ipAddress=1.2.3.4")
	assert.Contains(t, gotPath, "maxAgeInDays=90")
	assert.Equal(t, "test-key", gotKey)
	assert.Equal(t, "application/json", gotAccept)
}

func TestLookupVerdictDerivation(t *testing.T) {
	cases := []struct {
		name         string
		score        int
		totalReports int
		verdict      string
	}{
		{"zero reports → clean", 0, 0, "clean"},
		{"score 24 with reports → unknown", 24, 5, "unknown"},
		{"score 25 → suspicious", 25, 10, "suspicious"},
		{"score 74 → suspicious", 74, 10, "suspicious"},
		{"score 75 → malicious", 75, 10, "malicious"},
		{"score 100 → malicious", 100, 50, "malicious"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"data":{"abuseConfidenceScore":%d,"totalReports":%d}}`, tc.score, tc.totalReports)
			c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(body))
			})
			defer srv.Close()

			res, err := c.Lookup(context.Background(), "1.2.3.4")
			assert.Nil(t, err)
			assert.Equal(t, tc.verdict, res.Verdict)
		})
	}
}

func TestLookupScoreAndURL(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{"abuseConfidenceScore":87,"totalReports":42,"isp":"Test ISP","countryCode":"DE"}}`))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Nil(t, err)
	assert.Equal(t, "87/100", res.Score)
	assert.Equal(t, "https://www.abuseipdb.com/check/1.2.3.4", res.URL)
	assert.Contains(t, res.Summary, "87/100")
	assert.Contains(t, res.Summary, "Test ISP")
	assert.Contains(t, res.Summary, "DE")
}

func TestLookupCategories(t *testing.T) {
	body := `{"data":{"abuseConfidenceScore":80,"totalReports":5,"reports":[{"categories":[18,22]},{"categories":[14]}]}}`
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	defer srv.Close()

	res, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Nil(t, err)
	assert.Contains(t, res.Summary, "Brute-Force")
	assert.Contains(t, res.Summary, "SSH")
	assert.Contains(t, res.Summary, "Port Scan")
}

func TestLookupServerError(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := c.Lookup(context.Background(), "1.2.3.4")
	assert.Error(t, err)
}
