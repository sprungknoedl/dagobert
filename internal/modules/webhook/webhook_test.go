package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSupports(t *testing.T) {
	m := &Module{}

	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{"Case passes", model.Case{ID: "c1"}, true},
		{"Evidence passes", model.Evidence{Name: "x.evtx"}, true},
		{"Indicator passes", model.Indicator{Type: "IP", TLP: "TLP:GREEN"}, true},
		{"Indicator TLP:RED denied", model.Indicator{Type: "IP", TLP: "TLP:RED"}, false},
		{"unsupported type rejected", model.Malware{Hash: "x"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Supports(tc.obj))
		})
	}
}

func TestRun(t *testing.T) {
	t.Run("delivers envelope with headers and no signature when secret unset", func(t *testing.T) {
		var gotBody []byte
		var gotEvent, gotSig string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			gotEvent = r.Header.Get("X-Dagobert-Event")
			gotSig = r.Header.Get("X-Dagobert-Signature")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		m := &Module{}
		job := model.Job{
			Case:   model.Case{ID: "c1", Name: "Test Case"},
			Object: model.Object{Payload: model.Evidence{ID: "e1", Name: "x.evtx"}},
			Settings: map[string]string{
				"url":   srv.URL,
				"event": "evidence.added",
				"rule":  "myrule",
			},
		}

		err := m.Run(context.Background(), nil, job)
		assert.Nil(t, err)
		assert.Equal(t, "evidence.added", gotEvent)
		assert.Equal(t, "", gotSig)

		var env map[string]any
		assert.Nil(t, json.Unmarshal(gotBody, &env))
		assert.Equal(t, "evidence.added", env["event"])
		assert.Equal(t, "myrule", env["rule"])
		assert.NotNil(t, env["timestamp"])
		assert.NotNil(t, env["case"])
		assert.NotNil(t, env["object"])
	})

	t.Run("signs the body when secret is set", func(t *testing.T) {
		var gotBody []byte
		var gotSig string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			gotSig = r.Header.Get("X-Dagobert-Signature")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		m := &Module{secret: "s3cr3t"}
		job := model.Job{
			Case:     model.Case{ID: "c1"},
			Object:   model.Object{Payload: model.Case{ID: "c1"}},
			Settings: map[string]string{"url": srv.URL, "event": "case.added", "rule": "r"},
		}

		assert.Nil(t, m.Run(context.Background(), nil, job))

		mac := hmac.New(sha256.New, []byte("s3cr3t"))
		mac.Write(gotBody)
		want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		assert.Equal(t, want, gotSig)
	})

	t.Run("non-2xx response returns an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		m := &Module{}
		job := model.Job{
			Settings: map[string]string{"url": srv.URL, "event": "case.added", "rule": "r"},
		}

		err := m.Run(context.Background(), nil, job)
		assert.NotNil(t, err)
	})

	t.Run("empty URL returns an error without a request", func(t *testing.T) {
		m := &Module{}
		job := model.Job{Settings: map[string]string{"event": "case.added", "rule": "r"}}

		err := m.Run(context.Background(), nil, job)
		assert.NotNil(t, err)
	})
}
