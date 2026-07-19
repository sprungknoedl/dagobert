// Package webhook implements the Webhook automation module, which POSTs a JSON
// event to a per-rule URL when an automation rule fires.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/modules/utils"
)

type Module struct {
	secret string
}

func NewModule() *Module {
	return &Module{secret: os.Getenv("MODULE_WEBHOOK_SECRET")}
}

func (m *Module) Name() string { return "Webhook" }

func (m *Module) Description() string {
	return "Webhook POSTs a JSON event to a per-rule URL when an automation rule fires, so external automation (n8n, Zapier, scripts) can react to case/evidence/indicator changes."
}

func (m *Module) Supports(obj any) bool {
	switch v := obj.(type) {
	case model.Case:
		return true
	case model.Evidence:
		return true
	case model.Indicator:
		return v.TLP != "TLP:RED"
	default:
		return false
	}
}

func (m *Module) Validate() (model.Module, error) {
	return m, nil
}

type envelope struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Rule      string    `json:"rule"`
	Case      any       `json:"case"`
	Object    any       `json:"object"`
}

func (m *Module) Run(ctx context.Context, store *model.Store, job model.Job) error {
	url := job.Settings["url"]
	event := job.Settings["event"]
	rule := job.Settings["rule"]
	if url == "" {
		return errors.New("webhook: no URL configured on automation rule")
	}

	body, err := json.Marshal(envelope{
		Event:     event,
		Timestamp: time.Now(),
		Rule:      rule,
		Case:      job.Case,
		Object:    job.Object.Payload,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, utils.LookupTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dagobert-Event", event)
	if m.secret != "" {
		mac := hmac.New(sha256.New, []byte(m.secret))
		mac.Write(body)
		req.Header.Set("X-Dagobert-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook %s returned %s", url, resp.Status)
	}

	return nil
}

// RenderSettings is never rendered in practice: Webhook is excluded from the
// manual "Run module" list (see modules.Supported), the only caller of this
// method.
func (m *Module) RenderSettings() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error { return nil })
}
