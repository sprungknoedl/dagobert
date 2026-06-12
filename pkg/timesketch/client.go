// Package timesketch implements a client for the Timesketch HTTP API.
// The client that logs in lazily on first use and re-authenticates once
// when the session expires.
package timesketch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sprungknoedl/dagobert/pkg/fp"
	"golang.org/x/net/html"
)

var ErrNotFound = errors.New("timesketch: not found")
var ErrNotConfigured = errors.New("timesketch: integration is not configured")

// uploadChunkSize mirrors the official importer's slice size; a variable so
// tests can shrink it.
var uploadChunkSize = 50 * 1024 * 1024

type Config struct {
	URL           string
	Username      string
	Password      string
	SkipVerifyTLS bool
}

type Client struct {
	cfg    Config
	client *http.Client

	mu        sync.Mutex // guards csrfToken and loggedIn; serializes logins
	csrfToken string
	loggedIn  bool
}

type Response[T any] struct {
	Meta struct {
		CurrentPage int    `json:"current_page"`
		CurrentUser string `json:"current_user"`
		HasNext     bool   `json:"has_next"`
		HasPrev     bool   `json:"has_prev"`
		NextPage    string `json:"next_page"`
		PrevPage    string `json:"prev_page"`
		TotalItems  int    `json:"total_items"`
		TotalPages  int    `json:"total_pages"`

		Attributes Attributes `json:"attributes"`
		Mappings   []Field    `json:"mappings"`
	} `json:"meta"`
	Objects []T `json:"objects"`
}

type Attributes map[string]Attribute

type Attribute struct {
	Ontology string `json:"ontology"`
	Values   struct {
		Data []Intelligence `json:"data"`
	} `json:"value"`
}

type Intelligence struct {
	Type        string   `json:"type"`
	IOC         string   `json:"ioc"`
	Tags        []string `json:"tags"`
	ExternalURI string   `json:"externalURI"`
}

type Sketch struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	Timelines []Timeline `json:"timelines"`

	// copied over from meta
	Mappings   []Field    `json:"mappings"`
	Attributes Attributes `json:"attributes"`
}

type Timeline struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID     string         `json:"_id"`
	Index  string         `json:"_index"`
	Score  string         `json:"_score"`
	Source map[string]any `json:"_source"`

	// copied over from source
	Message       string
	Datetime      time.Time
	TimestampDesc string
}

type Filter struct {
	From    int     `json:"from,omitempty"`
	Size    int     `json:"size"`
	Indices []int   `json:"indices"`
	Order   string  `json:"order"`
	Chips   []Chip  `json:"chips"`
	Fields  []Field `json:"fields"`
}

type Chip struct {
	Type     string `json:"type"`
	Field    string `json:"field"`
	Value    string `json:"value"`
	Operator string `json:"operator"`
	Active   bool   `json:"active"`
}

type Field struct {
	Field string `json:"field"`
	Type  string `json:"type"`
}

var StarredEventsChip = Chip{
	Type:     "label",
	Field:    "label",
	Value:    "__ts_star",
	Operator: "must",
	Active:   true,
}

// statusError carries the HTTP status of a failed API call so do() can
// distinguish expired sessions (401/403) from other failures.
type statusError struct {
	Code int
	Body string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("timesketch: unexpected status %d: %s", e.Code, e.Body)
}

func NewClient(cfg Config) *Client {
	jar, _ := cookiejar.New(nil) // never fails with nil options

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.SkipVerifyTLS}

	cfg.URL = strings.TrimRight(cfg.URL, "/")
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Jar:       jar,
			Transport: tr,
			// generous because uploads move multi-GB artifacts; callers bound
			// individual requests with their context
			Timeout: 30 * time.Minute,
		},
	}
}

func (c *Client) Configured() bool {
	return c.cfg.URL != ""
}

// Login forces the lazy session establishment, verifying that the configured
// credentials work. Used by the worker module's Validate.
func (c *Client) Login(ctx context.Context) error {
	return c.ensureSession(ctx)
}

func (c *Client) ensureSession(ctx context.Context) error {
	if !c.Configured() {
		return ErrNotConfigured
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loggedIn {
		return nil
	}
	return c.login(ctx)
}

// login performs CSRF fetch + login POST + verification. c.mu must be held.
func (c *Client) login(ctx context.Context) error {
	token, err := c.csrf(ctx)
	if err != nil {
		return fmt.Errorf("timesketch: fetch csrf token: %w", err)
	}

	data := url.Values{
		"username":   []string{c.cfg.Username},
		"password":   []string{c.cfg.Password},
		"csrf_token": []string{token},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/login/?next=%2F", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("timesketch: login: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Timesketch answers bad credentials with a 200 login page, so verify
	// with a cheap authenticated probe.
	if err := c.probe(ctx); err != nil {
		return fmt.Errorf("timesketch: login failed (check credentials): %w", err)
	}

	// the login rotated the session, so the pre-login token is stale
	token, err = c.csrf(ctx)
	if err != nil {
		return fmt.Errorf("timesketch: fetch csrf token: %w", err)
	}

	c.csrfToken = token
	c.loggedIn = true
	return nil
}

// probe checks that the session is authenticated via /api/v1/users/me/.
func (c *Client) probe(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+"/api/v1/users/me/", nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return fmt.Errorf("not authenticated (status %s)", resp.Status)
	}
	return nil
}

// csrf scrapes the csrf-token meta tag from the login/index page. c.mu must
// be held (or the client not yet shared).
func (c *Client) csrf(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+"/", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	token := ""
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			d := fp.ToMap(n.Attr, func(attr html.Attribute) string { return attr.Key })
			if d["name"].Val == "csrf-token" {
				token = d["content"].Val
				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	if token == "" {
		return "", errors.New("no csrf-token meta tag found")
	}
	return token, nil
}

// relogin drops the current session and logs in again.
func (c *Client) relogin(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loggedIn = false
	return c.login(ctx)
}

func (c *Client) token() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.csrfToken
}

// do executes one API request: ensures a session, sets headers, closes the
// body, checks the status code, and decodes JSON into out. On 401/403 it
// re-logs in once and retries. body (if non-nil) is JSON-encoded.
func (c *Client) do(ctx context.Context, method string, path string, body any, out any) error {
	build := func() (*http.Request, error) {
		var rd io.Reader
		if body != nil {
			buf := &bytes.Buffer{}
			if err := json.NewEncoder(buf).Encode(body); err != nil {
				return nil, err
			}
			rd = buf
		}

		req, err := http.NewRequestWithContext(ctx, method, c.cfg.URL+path, rd)
		if err != nil {
			return nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
		return req, nil
	}

	return c.roundtrip(ctx, build, out)
}

func (c *Client) roundtrip(ctx context.Context, build func() (*http.Request, error), out any) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	err := c.attempt(build, out)
	var serr *statusError
	if errors.As(err, &serr) && (serr.Code == http.StatusUnauthorized || serr.Code == http.StatusForbidden) {
		if err := c.relogin(ctx); err != nil {
			return err
		}
		err = c.attempt(build, out)
	}
	return err
}

func (c *Client) attempt(build func() (*http.Request, error), out any) error {
	req, err := build()
	if err != nil {
		return err
	}
	req.Header.Set("X-CSRFToken", c.token())

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return &statusError{Code: resp.StatusCode, Body: strings.TrimSpace(string(excerpt))}
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) ListSketches(ctx context.Context) ([]Sketch, error) {
	carrier := Response[Sketch]{}
	err := c.do(ctx, http.MethodGet, "/api/v1/sketches/", nil, &carrier)
	return carrier.Objects, err
}

func (c *Client) GetSketch(ctx context.Context, id int) (Sketch, error) {
	carrier := Response[Sketch]{}
	if err := c.do(ctx, http.MethodGet, "/api/v1/sketches/"+strconv.Itoa(id), nil, &carrier); err != nil {
		return Sketch{}, err
	}
	if len(carrier.Objects) < 1 {
		return Sketch{}, ErrNotFound
	}

	sketch := carrier.Objects[0]
	sketch.Attributes = carrier.Meta.Attributes
	sketch.Mappings = carrier.Meta.Mappings
	return sketch, nil
}

// Explore runs one explore query with the filter as given (single page).
func (c *Client) Explore(ctx context.Context, id int, query string, filter Filter) ([]Event, error) {
	carrier, err := c.explore(ctx, id, query, filter)
	return carrier.Objects, err
}

// ExploreAll follows meta.has_next until the result set is exhausted.
func (c *Client) ExploreAll(ctx context.Context, id int, query string, filter Filter) ([]Event, error) {
	events := []Event{}
	for {
		carrier, err := c.explore(ctx, id, query, filter)
		if err != nil {
			return nil, err
		}

		events = append(events, carrier.Objects...)
		if !carrier.Meta.HasNext || len(carrier.Objects) == 0 {
			return events, nil
		}
		filter.From += len(carrier.Objects)
	}
}

func (c *Client) explore(ctx context.Context, id int, query string, filter Filter) (Response[Event], error) {
	carrier := Response[Event]{}
	err := c.do(ctx, http.MethodPost, "/api/v1/sketches/"+strconv.Itoa(id)+"/explore/", map[string]any{
		"query":  query,
		"filter": filter,
	}, &carrier)

	carrier.Objects = fp.Apply(carrier.Objects, func(obj Event) Event {
		datetime, _ := obj.Source["datetime"].(string)
		obj.Datetime, _ = time.Parse(time.RFC3339, datetime)
		obj.Message, _ = obj.Source["message"].(string)
		obj.TimestampDesc, _ = obj.Source["timestamp_desc"].(string)
		return obj
	})
	return carrier, err
}

// Upload streams the file to /api/v1/upload/ in uploadChunkSize slices using
// the chunk protocol of the official importer (constant memory; the final
// chunk triggers indexing server-side). The timeline is named after the file.
func (c *Client) Upload(ctx context.Context, sketch int, path string) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	fh, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fh.Close()

	stat, err := fh.Stat()
	if err != nil {
		return err
	}

	base := filepath.Base(path)
	total := stat.Size()
	chunks := max((total+int64(uploadChunkSize)-1)/int64(uploadChunkSize), 1)

	buf := make([]byte, uploadChunkSize)
	for index := range chunks {
		n, err := io.ReadFull(fh, buf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return err
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", base)
		if err != nil {
			return err
		}
		if _, err := part.Write(buf[:n]); err != nil {
			return err
		}

		if err := errors.Join(
			writer.WriteField("name", base),
			writer.WriteField("sketch_id", strconv.Itoa(sketch)),
			writer.WriteField("total_file_size", strconv.FormatInt(total, 10)),
			writer.WriteField("chunk_index", strconv.FormatInt(index, 10)),
			writer.WriteField("chunk_byte_offset", strconv.FormatInt(index*int64(uploadChunkSize), 10)),
			writer.WriteField("chunk_total_chunks", strconv.FormatInt(chunks, 10)),
			writer.WriteField("provider", "Dagobert"),
			writer.WriteField("context", "upload of dagobert evidence: "+base),
		); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}

		chunk := body.Bytes()
		build := func() (*http.Request, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/api/v1/upload/", bytes.NewReader(chunk))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())
			return req, nil
		}

		if err := c.roundtrip(ctx, build, nil); err != nil {
			return fmt.Errorf("timesketch: upload chunk %d/%d: %w", index+1, chunks, err)
		}
	}

	return nil
}
