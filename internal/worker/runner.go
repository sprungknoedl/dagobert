package worker

import (
	"archive/zip"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
)

var modules = []string{}

func StartWorker() {
	ValidateHayabusa()
	ValidatePlaso()
	ValidateTimesketch()
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
	ch := make(chan Job)
	for i := 0; i < num; i++ {
		go DispatchJob(ch)
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
	q.Add("modules", strings.Join(modules, ","))
	q.Add("workers", strconv.Itoa(num))
	req.URL.RawQuery = q.Encode()

	slog.Info("worker is ready", "upstream", os.Getenv("DAGOBERT_URL"), "modules", strings.Join(modules, ","))
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to send request", "err", err)
		return
	}

	dec := json.NewDecoder(resp.Body)
	for {
		job := Job{}
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

func DispatchJob(ch <-chan Job) {
	for job := range ch {
		var err error
		switch job.Name {
		case "keep-alive":
			slog.Debug("received keep-alive")
			return
		case "Hayabusa":
			err = RunHayabusa(job)
		case "Plaso (Windows Preset)":
			err = RunPlasoWindows(job)
		case "Plaso (Linux Preset)":
			err = RunPlasoLinux(job)
		case "Plaso (MacOS Preset)":
			err = RunPlasoMacOS(job)
		case "Plaso (Filesystem Timeline)":
			err = RunPlasoMFT(job)
		case "Timesketch Importer":
			err = UploadToTimesketch(job)
		default:
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

func AddFromFS(obj model.Evidence) error {
	body := bytes.NewBuffer(nil)
	form := multipart.NewWriter(body)

	err := errors.Join(
		form.WriteField("Name", obj.Name),
		form.WriteField("Type", obj.Type),
		form.WriteField("Source", obj.Source),
		form.WriteField("Notes", obj.Notes))
	if err != nil {
		return err
	}

	err = form.Close()
	if err != nil {
		return err
	}

	uri := os.Getenv("DAGOBERT_URL") + "/cases/" + obj.CaseID + "/evidences/new"
	req, err := http.NewRequest(http.MethodPost, uri, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("X-API-Key", os.Getenv("DAGOBERT_API_KEY"))
	client := http.Client{}
	_, err = client.Do(req)
	return err
}

func unpack(obj model.Evidence) (string, error) {
	src := filepath.Join("files", "evidences", obj.CaseID, obj.Name)
	reader, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	dir := filepath.Join("files", "tmp", random(10))
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	cleanup := func(err error) error {
		os.RemoveAll(dir) // try to cleanup but ignore if it fails
		return err
	}

	for _, file := range reader.File {
		dst := filepath.Clean(filepath.Join(dir, file.Name))

		// Check for file traversal attack
		if !strings.HasPrefix(dst, dir) {
			return "", cleanup(fmt.Errorf("invalid file path: %s", file.Name))
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, file.Mode()); err != nil {
				return "", cleanup(err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
				return "", cleanup(err)
			}

			destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return "", cleanup(err)
			}
			defer destFile.Close()

			srcFile, err := file.Open()
			if err != nil {
				return "", cleanup(err)
			}
			defer srcFile.Close()

			if _, err := io.Copy(destFile, srcFile); err != nil {
				return "", cleanup(err)
			}
		}
	}

	return dir + string(filepath.Separator), nil
}

func random(n int) string {
	// random string
	var src = rand.NewSource(time.Now().UnixNano())

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
