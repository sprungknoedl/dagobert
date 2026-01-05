package worker

import (
	"bytes"
	"cmp"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/worker/hayabusa"
	"github.com/sprungknoedl/dagobert/app/worker/plaso"
	"github.com/sprungknoedl/dagobert/app/worker/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

var Modules = map[string]model.Module{}

type Worker struct {
	WorkerID   string
	RemoteAddr string
	Modules    []string
	Workers    int
}

func init() {
	ctors := []func() model.Module{
		hayabusa.NewModule,
		plaso.NewModule,
		timesketch.NewModule,
	}
	for _, ctor := range ctors {
		m := ctor()
		Modules[m.Name()] = m
	}
}

func Supported(obj any) []model.Module {
	return fp.ToList(fp.FilterM(Modules, func(p model.Module) bool { return p.Supports(obj) }))
}

func Run(cmd *cobra.Command, args []string) {
	// validate modules, keep only modules definitions we can run
	modules := fp.FilterM(Modules, func(m model.Module) bool {
		_, err := m.Validate()
		return err == nil // TODO: error logging?
	})

	if len(modules) == 0 {
		slog.Error("worker not ready")
		return
	}

	// starting workers
	num, err := strconv.Atoi(cmp.Or(os.Getenv("DAGOBERT_WORKERS"), "3"))
	if len(modules) == 0 {
		slog.Error("invalid number of workers", "err", err)
		return
	}

	slog.Info("starting workers", "num", num)
	ch := make(chan model.Job)
	for i := 0; i < num; i++ {
		go DispatchJob(modules, ch)
	}

	// dagobert client
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig.InsecureSkipVerify = os.Getenv("DAGOBERT_SKIP_VERIFY_TLS") == "true"

	client := http.Client{
		Transport: tr,
	}
	req, err := http.NewRequest(http.MethodGet, os.Getenv("DAGOBERT_URL")+"/internal/jobs", nil)
	if err != nil {
		slog.Error("failed to create request", "err", err)
	}

	// set SSE specific headers
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("X-API-Key", os.Getenv("DAGOBERT_API_KEY"))

	q := req.URL.Query()
	q.Add("modules", strings.Join(fp.Keys(modules), ","))
	q.Add("workers", strconv.Itoa(num))
	req.URL.RawQuery = q.Encode()

	slog.Info("worker is ready", "upstream", os.Getenv("DAGOBERT_URL"), "modules", strings.Join(fp.Keys(modules), ","))
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to send request", "err", err)
		return
	}

	dec := json.NewDecoder(resp.Body)
	for {
		job := model.Job{}
		err = dec.Decode(&job)
		if err != nil {
			slog.Error("failed to decode job", "err", err)
			return
		}

		slog.Info("received job", "job", job)
		job.Ctx = req.Context()
		ch <- job
	}
}

func DispatchJob(modules map[string]model.Module, ch <-chan model.Job) {
	for job := range ch {
		var err error
		if job.Name == "keep-alive" {
			slog.Debug("received keep-alive")
			continue
		}

		if m, ok := modules[job.Name]; ok {
			err = m.Run(job)
		} else {
			slog.Error("unknown module name", "job", job.ID, "module", job.Name)
			continue
		}

		errmsg := ""
		if err != nil {
			errmsg = err.Error()
			slog.Warn("failed to process job", "job", job.ID, "module", job.Name, "err", err)
		}

		err = AckJob(model.Job{
			ID:     job.ID,
			Status: fp.If(err != nil, "Failed", "Success"),
			Error:  errmsg,
		})
		if err != nil {
			slog.Warn("failed to ack job", "job", job.ID, "module", job.Name, "err", err)
		}
	}
}

func AckJob(job model.Job) error {
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(job)
	if err != nil {
		return err
	}

	uri := os.Getenv("DAGOBERT_URL") + "/internal/jobs/ack"
	req, err := http.NewRequest(http.MethodPost, uri, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", os.Getenv("DAGOBERT_API_KEY"))
	client := http.Client{}
	_, err = client.Do(req)
	return err
}
