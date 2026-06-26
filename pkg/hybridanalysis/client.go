// Package hybridanalysis implements a small client for the Hybrid Analysis
// (Falcon Sandbox) v2 API. It looks up a file hash and returns a distilled
// result (verdict, score, summary, deep link) rather than the raw API response.
package hybridanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	Score     string // "<threat_score>/100" or empty
	Summary   string // human-readable multi-line prose
	URL       string // deep link to HA search result; empty when no record
	FetchedAt time.Time
}

// report is one entry from the /search/hash response array.
type report struct {
	Verdict                string `json:"verdict"`
	ThreatScore            *int   `json:"threat_score"`
	VXFamily               string `json:"vx_family"`
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
	io.Copy(io.Discard, resp.Body)

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

	var reports []report
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		return Result{}, fmt.Errorf("hybridanalysis: decode response: %w", err)
	}

	return distill(reports, hash, now), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("api-key", c.cfg.APIKey)
	req.Header.Set("User-Agent", "Falcon Sandbox")
	req.Header.Set("Accept", "application/json")
}

func distill(reports []report, hash string, now time.Time) Result {
	if len(reports) == 0 {
		return Result{
			Verdict:   "unknown",
			Summary:   "No analysis found in Hybrid Analysis\nFetched: " + now.Format("2006-01-02 15:04 MST"),
			FetchedAt: now,
		}
	}

	// Pick the most relevant report by highest threat_score.
	best := reports[0]
	for _, r := range reports[1:] {
		if r.ThreatScore != nil && (best.ThreatScore == nil || *r.ThreatScore > *best.ThreatScore) {
			best = r
		}
	}

	verdict := mapVerdict(best.Verdict)

	score := ""
	if best.ThreatScore != nil {
		score = fmt.Sprintf("%d/100", *best.ThreatScore)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Verdict: %s", verdict)
	if score != "" {
		fmt.Fprintf(&b, " (score: %s)", score)
	}
	b.WriteString("\n")
	if best.VXFamily != "" {
		fmt.Fprintf(&b, "Family: %s\n", best.VXFamily)
	}
	if best.EnvironmentDescription != "" {
		fmt.Fprintf(&b, "Environment: %s\n", best.EnvironmentDescription)
	}
	fmt.Fprintf(&b, "Fetched: %s", now.Format("2006-01-02 15:04 MST"))

	link := linkBase + url.QueryEscape(hash)

	return Result{
		Verdict:   verdict,
		Score:     score,
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
