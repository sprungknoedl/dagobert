// Package hybridanalysis implements a small client for the Hybrid Analysis
// (Falcon Sandbox) v2 API. It looks up a file hash and returns a distilled
// result (verdict, summary, deep link) rather than the raw API response.
package hybridanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
)

const (
	// apiBase uses the canonical host without "www": www.hybrid-analysis.com
	// 301-redirects to it, so this avoids a needless redirect hop per request.
	apiBase  = "https://hybrid-analysis.com/api/v2"
	linkBase = "https://www.hybrid-analysis.com/search?query="
)

type Config struct {
	APIKey string
}

type Client struct {
	cfg     Config
	client  *http.Client
	baseURL string
}

// Result is the distilled lookup outcome written onto the indicator.
type Result struct {
	Verdict   string // malicious | suspicious | clean | unknown
	Summary   string // human-readable multi-line prose
	URL       string // deep link to HA search result; empty when no record
	FetchedAt time.Time
}

// report is one entry of the /search/hash "reports" list. Verdict is null
// for analyses that errored.
type report struct {
	Verdict                string `json:"verdict"`
	EnvironmentDescription string `json:"environment_description"`
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		client:  &http.Client{},
		baseURL: apiBase,
	}
}

func (c *Client) Configured() bool { return c.cfg.APIKey != "" }

// Verify confirms the API key by querying the key info endpoint.
func (c *Client) Verify(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/key/current", nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		slog.Warn("hybridanalysis: failed to drain response body", "err", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("hybridanalysis: authentication failed (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("hybridanalysis: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Lookup queries a hash against the Hybrid Analysis /search/hash endpoint.
// The hash is passed as a query parameter on a GET request: the POST form
// variant of this endpoint was deprecated in API v2.35.0 (returns 410).
func (c *Client) Lookup(ctx context.Context, hash string) (Result, error) {
	now := time.Now().UTC()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/search/hash?hash="+url.QueryEscape(hash), nil)
	if err != nil {
		return Result{}, err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	// An unknown hash returns 404 ("Requested hash not found"). Treat it like an
	// empty result: a clean "unknown" verdict, not a job failure.
	if resp.StatusCode == http.StatusNotFound {
		return distill(nil, hash, now), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Result{}, fmt.Errorf("hybridanalysis: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(excerpt)))
	}

	var ar struct {
		Reports []report `json:"reports"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return Result{}, fmt.Errorf("hybridanalysis: decode response: %w", err)
	}

	return distill(ar.Reports, hash, now), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("api-key", c.cfg.APIKey)
	req.Header.Set("User-Agent", "Falcon Sandbox")
	req.Header.Set("Accept", "application/json")
}

func distill(reports []report, hash string, now time.Time) Result {
	reports = slices.DeleteFunc(reports, func(r report) bool { return r.Verdict == "" })
	if len(reports) == 0 {
		return Result{
			Verdict:   "unknown",
			Summary:   "No analysis found in Hybrid Analysis\nFetched: " + now.Format("2006-01-02 15:04 MST"),
			FetchedAt: now,
		}
	}

	best := slices.MaxFunc(reports, func(a, b report) int {
		ra, _ := model.VerdictSeverity(mapVerdict(a.Verdict))
		rb, _ := model.VerdictSeverity(mapVerdict(b.Verdict))
		return ra - rb
	})
	verdict := mapVerdict(best.Verdict)

	var b strings.Builder
	fmt.Fprintf(&b, "Verdict: %s\n", verdict)
	if best.EnvironmentDescription != "" {
		fmt.Fprintf(&b, "Environment: %s\n", best.EnvironmentDescription)
	}
	fmt.Fprintf(&b, "Fetched: %s", now.Format("2006-01-02 15:04 MST"))

	link := linkBase + url.QueryEscape(hash)

	return Result{
		Verdict:   verdict,
		Summary:   b.String(),
		URL:       link,
		FetchedAt: now,
	}
}

func mapVerdict(haVerdict string) string {
	switch haVerdict {
	case "malicious":
		return "malicious"
	case "suspicious":
		return "suspicious"
	case "no specific threat", "whitelisted":
		return "clean"
	default:
		return "unknown"
	}
}
