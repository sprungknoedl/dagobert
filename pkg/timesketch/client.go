package timesketch

import (
	"bytes"
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

	"github.com/sprungknoedl/dagobert/internal/fp"
	"golang.org/x/net/html"
)

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
	} `json:"meta"`
	Objects []T `json:"objects"`
}

type Sketch struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	User         string `json:"user"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	LastActivity string `json:"last_activity"`
}

func NewClient(uri, username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &Client{
		BaseURL:  uri,
		Username: username,
		Password: password,

		client: &http.Client{
			Jar: jar,
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

func (client Client) GetSketch(id int) ([]Sketch, error) { return nil, nil }

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
