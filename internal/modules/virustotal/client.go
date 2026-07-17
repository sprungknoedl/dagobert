// Package virustotal implements a small client for the VirusTotal v3 API.
// It looks up an indicator's value and returns a distilled result (verdict,
// score, summary, deep link) rather than the raw API response.
package virustotal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	// apiBase is the VirusTotal v3 REST base; linkBase is the human-facing
	// search deep link (matches the old external-lookup button).
	apiBase  = "https://www.virustotal.com/api/v3"
	linkBase = "https://www.virustotal.com/gui/search/"
)

// maxDetections caps how many distinct detection names fold into the summary.
const maxDetections = 5

type Config struct {
	APIKey string
}

type Client struct {
	cfg    Config
	client *http.Client
	// baseURL is apiBase in production; tests point it at an httptest server.
	baseURL string
}

// Result is the distilled lookup outcome written onto the indicator.
type Result struct {
	Verdict   string // malicious | suspicious | clean | unknown
	Score     string // detection ratio "<malicious>/<total>"
	Summary   string // human-readable multi-line prose
	URL       string // deep link; empty when there is none (e.g. no record)
	FetchedAt time.Time
}

// apiResponse is the subset of the v3 object envelope we read.
type apiResponse struct {
	Data struct {
		Attributes struct {
			LastAnalysisStats struct {
				Malicious  int `json:"malicious"`
				Suspicious int `json:"suspicious"`
				Harmless   int `json:"harmless"`
				Undetected int `json:"undetected"`
			} `json:"last_analysis_stats"`
			LastAnalysisResults map[string]struct {
				Category string `json:"category"`
				Result   string `json:"result"`
			} `json:"last_analysis_results"`
		} `json:"attributes"`
	} `json:"data"`
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		client:  &http.Client{},
		baseURL: apiBase,
	}
}

func (c *Client) Configured() bool { return c.cfg.APIKey != "" }

// Verify confirms the API key authenticates by fetching the key's own user
// object. 401/403 means a bad key; any other non-2xx or transport error is a
// connectivity failure. Used by the worker module's Validate.
func (c *Client) Verify(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/users/"+url.PathEscape(c.cfg.APIKey), nil)
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
		slog.Warn("virustotal: failed to drain response body", "err", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("virustotal: authentication failed (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("virustotal: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Lookup queries value against the endpoint for typ and distills the response.
// A 404 returns an "unknown" Result (no error) whose summary notes the missing
// record; transport errors, auth failures, 5xx, and malformed bodies return an
// error.
func (c *Client) Lookup(ctx context.Context, typ, value string) (Result, error) {
	now := time.Now().UTC()

	endpoint, err := endpoint(typ, value)
	if err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Result{
			Verdict:   "unknown",
			Summary:   "No record found in VirusTotal\nFetched: " + now.Format("2006-01-02 15:04 MST"),
			FetchedAt: now,
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Result{}, fmt.Errorf("virustotal: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(excerpt)))
	}

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return Result{}, fmt.Errorf("virustotal: decode response: %w", err)
	}

	return distill(ar, value, now), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("x-apikey", c.cfg.APIKey)
	req.Header.Set("Accept", "application/json")
}

// endpoint maps an indicator type to its v3 path. URLs use VT's identifier
// scheme: URL-safe base64 of the value with trailing '=' padding stripped.
func endpoint(typ, value string) (string, error) {
	switch typ {
	case "Hash":
		return "/files/" + url.PathEscape(value), nil
	case "IP":
		return "/ip_addresses/" + url.PathEscape(value), nil
	case "Domain":
		return "/domains/" + url.PathEscape(value), nil
	case "URL":
		return "/urls/" + base64.RawURLEncoding.EncodeToString([]byte(value)), nil
	default:
		return "", fmt.Errorf("virustotal: unsupported indicator type %q", typ)
	}
}

func distill(ar apiResponse, value string, now time.Time) Result {
	s := ar.Data.Attributes.LastAnalysisStats
	total := s.Malicious + s.Suspicious + s.Harmless + s.Undetected

	verdict := "unknown"
	switch {
	case s.Malicious > 0:
		verdict = "malicious"
	case s.Suspicious > 0:
		verdict = "suspicious"
	case total > 0:
		verdict = "clean"
	}

	score := fmt.Sprintf("%d/%d", s.Malicious, total)

	// distinct detection names from engines that flagged the value, sorted for
	// a deterministic summary (the results map iterates in random order)
	seen := map[string]bool{}
	names := []string{}
	for _, r := range ar.Data.Attributes.LastAnalysisResults {
		if (r.Category == "malicious" || r.Category == "suspicious") && r.Result != "" && !seen[r.Result] {
			seen[r.Result] = true
			names = append(names, r.Result)
		}
	}
	sort.Strings(names)
	if len(names) > maxDetections {
		names = names[:maxDetections]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Verdict: %s (%s)\n", verdict, score)
	if len(names) > 0 {
		fmt.Fprintf(&b, "Top detections: %s\n", strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "Fetched: %s", now.Format("2006-01-02 15:04 MST"))

	return Result{
		Verdict:   verdict,
		Score:     score,
		Summary:   b.String(),
		URL:       linkBase + value,
		FetchedAt: now,
	}
}
