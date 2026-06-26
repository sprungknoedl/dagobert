// Package abuseipdb implements a small client for the AbuseIPDB v2 API.
// It looks up an IP indicator and returns a distilled result (verdict,
// score, summary, deep link) rather than the raw API response.
package abuseipdb

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
	apiBase  = "https://api.abuseipdb.com/api/v2"
	linkBase = "https://www.abuseipdb.com/check/"

	ThresholdMalicious  = 75
	ThresholdSuspicious = 25

	maxCategories = 5
)

// categoryNames maps AbuseIPDB category IDs to human-readable labels.
var categoryNames = map[int]string{
	1:  "DNS Compromise",
	2:  "DNS Poisoning",
	3:  "Fraud Orders",
	4:  "DDoS Attack",
	5:  "FTP Brute-Force",
	6:  "Ping of Death",
	7:  "Phishing",
	8:  "Fraud VoIP",
	9:  "Open Proxy",
	10: "Web Spam",
	11: "Email Spam",
	12: "Blog Spam",
	13: "VPN IP",
	14: "Port Scan",
	15: "Hacking",
	16: "SQL Injection",
	17: "Spoofing",
	18: "Brute-Force",
	19: "Bad Web Bot",
	20: "Exploited Host",
	21: "Web App Attack",
	22: "SSH",
	23: "IoT Targeted",
}

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
	Score     string // "<score>/100"
	Summary   string // human-readable multi-line prose
	URL       string // deep link to the AbuseIPDB IP record
	FetchedAt time.Time
}

// apiResponse is the subset of the v2 check response we read.
type apiResponse struct {
	Data struct {
		AbuseConfidenceScore int    `json:"abuseConfidenceScore"`
		TotalReports         int    `json:"totalReports"`
		LastReportedAt       string `json:"lastReportedAt"`
		ISP                  string `json:"isp"`
		CountryCode          string `json:"countryCode"`
		Reports              []struct {
			Categories []int `json:"categories"`
		} `json:"reports"`
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

// Verify confirms the API key by doing a check on a known-benign IP.
func (c *Client) Verify(ctx context.Context) error {
	// Use a minimal check to validate the key without quota cost.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/check?ipAddress=8.8.8.8&maxAgeInDays=1", nil)
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
		return fmt.Errorf("abuseipdb: authentication failed (status %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("abuseipdb: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Lookup queries an IP address against the AbuseIPDB v2 /check endpoint.
func (c *Client) Lookup(ctx context.Context, ip string) (Result, error) {
	now := time.Now().UTC()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/check?ipAddress="+url.QueryEscape(ip)+"&maxAgeInDays=90", nil)
	if err != nil {
		return Result{}, err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Result{}, fmt.Errorf("abuseipdb: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(excerpt)))
	}

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return Result{}, fmt.Errorf("abuseipdb: decode response: %w", err)
	}

	return distill(ar, ip, now), nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Key", c.cfg.APIKey)
	req.Header.Set("Accept", "application/json")
}

func distill(ar apiResponse, ip string, now time.Time) Result {
	d := ar.Data
	score := d.AbuseConfidenceScore

	verdict := "unknown"
	switch {
	case score >= ThresholdMalicious:
		verdict = "malicious"
	case score >= ThresholdSuspicious:
		verdict = "suspicious"
	case d.TotalReports == 0:
		verdict = "clean"
	}

	// Collect distinct category names from all reports.
	seen := map[int]bool{}
	names := []string{}
	for _, r := range d.Reports {
		for _, cat := range r.Categories {
			if !seen[cat] {
				seen[cat] = true
				if name, ok := categoryNames[cat]; ok {
					names = append(names, name)
				} else {
					names = append(names, fmt.Sprintf("Category %d", cat))
				}
			}
			if len(names) >= maxCategories {
				break
			}
		}
		if len(names) >= maxCategories {
			break
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Score: %d/100 (%d reports)\n", score, d.TotalReports)
	if d.ISP != "" || d.CountryCode != "" {
		fmt.Fprintf(&b, "ISP: %s (%s)\n", d.ISP, d.CountryCode)
	}
	if d.LastReportedAt != "" {
		fmt.Fprintf(&b, "Last reported: %s\n", d.LastReportedAt[:10])
	}
	if len(names) > 0 {
		fmt.Fprintf(&b, "Categories: %s\n", strings.Join(names, ", "))
	}
	fmt.Fprintf(&b, "Fetched: %s", now.Format("2006-01-02 15:04 MST"))

	return Result{
		Verdict:   verdict,
		Score:     fmt.Sprintf("%d/100", score),
		Summary:   b.String(),
		URL:       linkBase + ip,
		FetchedAt: now,
	}
}
