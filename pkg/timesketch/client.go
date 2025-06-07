package timesketch

import (
	"bytes"
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
	"time"

	"github.com/sprungknoedl/dagobert/pkg/fp"
	"golang.org/x/net/html"
)

var ErrNoRows = errors.New("timesketch: no rows in result set")

type Client struct {
	BaseURL  string
	Username string
	Password string

	client    *http.Client
	csrfToken string
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

		Attributes map[string]struct {
			Ontology string `json:"ontology"`
			Values   struct {
				Data []struct {
					Type        string   `json:"type"`
					IOC         string   `json:"ioc"`
					Tags        []string `json:"tags"`
					ExternalURI string   `json:"externalURI"`
				} `json:"data"`
			} `json:"value"`
		} `json:"attributes"`

		Mappings []Field `json:"mappings"`
	} `json:"meta"`
	Objects []T `json:"objects"`
}

type Sketch struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// User         string `json:"user"`
	// Status       string `json:"status"`
	// CreatedAt    string `json:"created_at"`
	// LastActivity string `json:"last_activity"`

	Timelines []Timeline `json:"timelines"`

	// copied over from meta
	Mappings   []Field `json:"mappings"`
	Attributes map[string]struct {
		Ontology string `json:"ontology"`
		Values   struct {
			Data []struct {
				Type        string   `json:"type"`
				IOC         string   `json:"ioc"`
				Tags        []string `json:"tags"`
				ExternalURI string   `json:"externalURI"`
			} `json:"data"`
		} `json:"value"`
	} `json:"attributes"`
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

func NewClient(uri, username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: os.Getenv("TIMESKETCH_SKIP_VERIFY_TLS") == "true"}

	client := &Client{
		BaseURL:  uri,
		Username: username,
		Password: password,

		client: &http.Client{
			Jar:       jar,
			Transport: tr,
		},
	}

	client.csrf()
	data := url.Values{
		"username":   []string{client.Username},
		"password":   []string{client.Password},
		"csrf_token": []string{client.csrfToken},
	}
	resp, err := client.client.PostForm(client.BaseURL+"/login/?next=%2F", data)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return client, nil
}

func (client *Client) csrf() {
	req, err := http.NewRequest(http.MethodGet, client.BaseURL+"/", nil)
	if err != nil {
		return
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			d := fp.ToMap(n.Attr, func(attr html.Attribute) string { return attr.Key })
			if d["name"].Val == "csrf-token" {
				client.csrfToken = d["content"].Val
				return
			}
		}

		// traverse the child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// make a recursive call to your function
	traverse(doc)
}

func (client Client) ListSketches() ([]Sketch, error) {
	resp, err := client.client.Get(client.BaseURL + "/api/v1/sketches/")
	if err != nil {
		return nil, err
	}

	carrier := Response[Sketch]{}
	err = json.NewDecoder(resp.Body).Decode(&carrier)
	return carrier.Objects, err
}

func (client Client) GetSketch(id int) (Sketch, error) {
	resp, err := client.client.Get(client.BaseURL + "/api/v1/sketches/" + strconv.Itoa(id))
	if err != nil {
		return Sketch{}, err
	}

	carrier := Response[Sketch]{}
	err = json.NewDecoder(resp.Body).Decode(&carrier)
	if len(carrier.Objects) < 1 {
		return Sketch{}, ErrNoRows
	}

	sketch := carrier.Objects[0]
	sketch.Attributes = carrier.Meta.Attributes
	sketch.Mappings = carrier.Meta.Mappings
	return sketch, err
}

func (client Client) Explore(id int, query string, filter Filter) ([]Event, error) {
	client.csrf()

	body := &bytes.Buffer{}
	err := json.NewEncoder(body).Encode(map[string]any{
		"query":  query,
		"filter": filter,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, client.BaseURL+"/api/v1/sketches/"+strconv.Itoa(id)+"/explore/", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-CSRFToken", client.csrfToken)
	resp, err := client.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to explore: %s", resp.Status)
	}

	carrier := Response[Event]{}
	err = json.NewDecoder(resp.Body).Decode(&carrier)
	return fp.Apply(carrier.Objects, func(obj Event) Event {
		datetime, _ := obj.Source["datetime"].(string)
		obj.Datetime, _ = time.Parse(time.RFC3339, datetime)
		obj.Message, _ = obj.Source["message"].(string)
		obj.TimestampDesc, _ = obj.Source["timestamp_desc"].(string)
		return obj
	}), err
}

func (client Client) Upload(sketch int, path string) error {
	client.csrf()

	base := filepath.Base(path)
	fh, err := os.Open(path)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", base)
	if err != nil {
		return err
	}

	size, err := io.Copy(part, fh)
	if err != nil {
		return err
	}

	if err := errors.Join(
		writer.WriteField("name", strings.TrimSuffix(base, filepath.Ext(base))),
		writer.WriteField("sketch_id", strconv.FormatInt(int64(sketch), 10)),
		writer.WriteField("total_file_size", strconv.FormatInt(size, 10)),
		writer.WriteField("provider", "Dagobert"),
	); err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, client.BaseURL+"/api/v1/upload/", body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-CSRFToken", client.csrfToken)
	resp, err := client.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload file: %s", resp.Status)
	}

	return nil
}
