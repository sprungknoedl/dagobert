//go:build integration

// Smoke test against a live Timesketch instance, driven by the TIMESKETCH_*
// environment variables. Skips when they are unset. Run with:
//
//	go test -tags=integration ./pkg/timesketch/...

package timesketch

import (
	"os"
	"testing"
)

func TestIntegrationSmoke(t *testing.T) {
	url := os.Getenv("TIMESKETCH_URL")
	if url == "" {
		t.Skip("TIMESKETCH_URL not set, skipping integration test")
	}

	c := NewClient(Config{
		URL:           url,
		Username:      os.Getenv("TIMESKETCH_USER"),
		Password:      os.Getenv("TIMESKETCH_PASS"),
		SkipVerifyTLS: os.Getenv("TIMESKETCH_SKIP_VERIFY_TLS") == "true",
	})

	if err := c.Login(t.Context()); err != nil {
		t.Fatal(err)
	}

	sketches, err := c.ListSketches(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found %d sketches", len(sketches))
}
