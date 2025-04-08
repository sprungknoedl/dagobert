package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker"
)

type Module struct {
	Name        string
	Description string
	Status      string
	Error       string
}

type Worker struct {
	WorkerID   string
	RemoteAddr string
	Modules    []string
	Workers    int
}

type JobCtrl struct {
	store *model.Store
	acl   *ACL

	workermu sync.Mutex
	workers  map[string]Worker
}

func NewJobCtrl(store *model.Store, acl *ACL) *JobCtrl {
	return &JobCtrl{
		store:   store,
		acl:     acl,
		workers: make(map[string]Worker),
	}
}

func (ctrl JobCtrl) Workers() []Worker {
	ctrl.workermu.Lock()
	defer ctrl.workermu.Unlock()

	workers := make([]Worker, 0, len(ctrl.workers))
	for _, worker := range ctrl.workers {
		workers = append(workers, worker)
	}
	return workers
}

func (ctrl JobCtrl) ListMods(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := ctrl.store.GetEvidence(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	modules := worker.Supported(obj)
	jobs, err := ctrl.store.GetJobs(obj.ID)
	if err != nil {
		Err(w, r, err)
		return
	}

	m := fp.ToMap(jobs, func(obj model.Job) string { return obj.Name })
	runs := fp.Apply(modules, func(obj worker.Module) Module {
		return Module{
			Name:        obj.Name,
			Description: obj.Description,
			Status:      m[obj.Name].Status,
			Error:       m[obj.Name].Error,
		}
	})

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/evidences-process.html", map[string]any{
		"obj":  obj,
		"runs": runs,
	})
}

func (ctrl JobCtrl) PopJob(w http.ResponseWriter, r *http.Request) {
	// set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// create a channel for client disconnection
	gone := r.Context().Done()

	// create polling ticker
	rc := http.NewResponseController(w)
	t := time.NewTicker(time.Second)
	defer t.Stop()

	ka := time.NewTicker(time.Minute)
	defer ka.Stop()

	// create worker id
	workerid := fp.Random(20)
	modules := strings.Split(r.URL.Query().Get("modules"), ",")
	workers, _ := strconv.Atoi(r.URL.Query().Get("workers"))
	log.Printf("worker %q started", workerid)

	// register worker
	ctrl.workermu.Lock()
	ctrl.workers[workerid] = Worker{
		WorkerID:   workerid,
		RemoteAddr: r.RemoteAddr,
		Modules:    modules,
		Workers:    workers,
	}
	ctrl.workermu.Unlock()

	for {
		select {
		case <-gone:
			log.Println("client disconnected")
			goto cleanup

		case <-ka.C:
			if err := sendJob(w, rc, worker.Job{
				Name:        "keep-alive",
				WorkerToken: workerid,
			}); err != nil {
				log.Printf("%v", err)
				goto cleanup
			}

		case <-t.C:
			job, err := ctrl.store.PopJob(workerid, modules)
			if err != nil && err != sql.ErrNoRows {
				log.Printf("error fetching job: %v", err)
				goto cleanup
			}
			if err == sql.ErrNoRows {
				continue
			}

			// fetch objects
			evidence, err1 := ctrl.store.GetEvidence(job.CaseID, job.EvidenceID)
			kase, err2 := ctrl.store.GetCase(job.CaseID)
			if err := errors.Join(err1, err2); err != nil {
				log.Printf("error encoding job: %v", err)
				goto cleanup
			}

			if err := sendJob(w, rc, worker.Job{
				ID:          job.ID,
				WorkerToken: workerid,
				Name:        job.Name,
				Case:        kase,
				Evidence:    evidence,
			}); err != nil {
				log.Printf("%v", err)
				goto cleanup
			}
		}
	}

cleanup:
	ctrl.workermu.Lock()
	delete(ctrl.workers, workerid)
	ctrl.workermu.Unlock()

	log.Printf("worker %q quit", workerid)
	err := ctrl.store.RescheduleWorkerJobs(workerid)
	if err != nil {
		log.Printf("error rescheduling jobs for %q: %v", workerid, err)
	}
}

func sendJob(w http.ResponseWriter, rc *http.ResponseController, job worker.Job) error {
	err := json.NewEncoder(w).Encode(job)
	if err != nil {
		return fmt.Errorf("error encoding job: %w", err)
	}

	err = rc.Flush()
	if err != nil {
		return fmt.Errorf("error flushing job: %w", err)
	}

	return nil
}

func (ctrl JobCtrl) AckJob(w http.ResponseWriter, r *http.Request) {
	dto := model.Job{}
	if err := Decode(r, &dto); err != nil {
		Err(w, r, err)
		return
	}

	err := ctrl.store.AckJob(dto.ID, dto.Status, dto.Error)
	if err != nil {
		Err(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ctrl JobCtrl) PushJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	name := r.FormValue("name")
	obj, err := ctrl.store.GetEvidence(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	err = ctrl.store.PushJob(model.Job{
		ID:         fp.Random(10),
		CaseID:     cid,
		EvidenceID: id,
		Name:       name,
		Status:     "Scheduled",
	})
	if err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "evidence:"+obj.ID, "Run extension %q on evidence %q", name, obj.Name)
	ctrl.ListMods(w, r)
}
