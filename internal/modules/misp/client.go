// Package misp implements a small client for the MISP REST API. It looks up
// an indicator's value via attributes/restSearch and returns a distilled
// result (verdict, summary, deep link) rather than the raw API response.
package misp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxTags = 5

var threatLevels = map[int]string{1: "High", 2: "Medium", 3: "Low", 4: "Undefined"}

type Config struct {
	URL           string
	APIKey        string
	SkipVerifyTLS bool
}

type Client struct {
	cfg    Config
	client *http.Client
}

// Result is the distilled lookup outcome written onto the indicator.
type Result struct {
	Verdict   string // malicious | suspicious | clean | unknown
	Summary   string // human-readable multi-line prose
	URL       string // deep link to the deciding event; empty when there is no match
	FetchedAt time.Time
}

// apiResponse is the subset of the attributes/restSearch response we read.
type apiResponse struct {
	Response struct {
		Attribute []struct {
			EventID string `json:"event_id"`
			ToIDs   bool   `json:"to_ids"`
			Event   struct {
				ThreatLevelID string `json:"threat_level_id"`
			} `json:"Event"`
			Tag []struct {
				Name string `json:"name"`
			} `json:"Tag"`
		} `json:"Attribute"`
	} `json:"response"`
}

func NewClient(cfg Config) *Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.SkipVerifyTLS}

	cfg.URL = strings.TrimRight(cfg.URL, "/")
	return &Client{
		cfg:    cfg,
		client: &http.Client{Transport: tr},
	}
}

func (c *Client) Configured() bool { return c.cfg.URL != "" && c.cfg.APIKey != "" }

// Verify confirms the key authenticates by fetching the instance version.
func (c *Client) Verify(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+"/servers/getVersion", nil)
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
		slog.Warn("misp: failed to drain response body", "err", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("misp: authentication failed (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("misp: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Lookup searches MISP attributes for an exact value match and distills the
// matches. No match is an "unknown" Result, not an error.
func (c *Client) Lookup(ctx context.Context, value string) (Result, error) {
	now := time.Now().UTC()

	body, err := json.Marshal(map[string]any{
		"returnFormat":     "json",
		"value":            value,
		"includeContext":   true,
		"includeEventTags": true,
	})
	if err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/attributes/restSearch", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Result{}, fmt.Errorf("misp: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(excerpt)))
	}

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return Result{}, fmt.Errorf("misp: decode response: %w", err)
	}

	return c.distill(ar, now), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", c.cfg.APIKey)
	req.Header.Set("Accept", "application/json")
}

func (c *Client) distill(ar apiResponse, now time.Time) Result {
	fetched := "Fetched: " + now.Format("2006-01-02 15:04 MST")
	attrs := ar.Response.Attribute
	if len(attrs) == 0 {
		return Result{
			Verdict:   "unknown",
			Summary:   "No record found in MISP\n" + fetched,
			FetchedAt: now,
		}
	}

	// the deciding attribute: to_ids first, then the most severe event threat level
	best, bestLevel := -1, 0
	highest := 4
	events := map[string]bool{}
	seen := map[string]bool{}
	tags := []string{}
	for i, a := range attrs {
		level, err := strconv.Atoi(a.Event.ThreatLevelID)
		if err != nil || threatLevels[level] == "" {
			level = 4
		}
		highest = min(highest, level)
		events[a.EventID] = true
		for _, t := range a.Tag {
			if !seen[t.Name] {
				seen[t.Name] = true
				tags = append(tags, t.Name)
			}
		}

		if best < 0 || (a.ToIDs && !attrs[best].ToIDs) || (a.ToIDs == attrs[best].ToIDs && level < bestLevel) {
			best, bestLevel = i, level
		}
	}
	sort.Strings(tags)
	if len(tags) > maxTags {
		tags = tags[:maxTags]
	}

	verdict := "clean"
	switch {
	case attrs[best].ToIDs && bestLevel <= 2:
		verdict = "malicious"
	case attrs[best].ToIDs:
		verdict = "suspicious"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Verdict: %s\n", verdict)
	fmt.Fprintf(&b, "Matches: %d attributes in %d events\n", len(attrs), len(events))
	fmt.Fprintf(&b, "Threat level: %s\n", threatLevels[highest])
	if len(tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(tags, ", "))
	}
	b.WriteString(fetched)

	return Result{
		Verdict:   verdict,
		Summary:   b.String(),
		URL:       c.cfg.URL + "/events/view/" + attrs[best].EventID,
		FetchedAt: now,
	}
}
