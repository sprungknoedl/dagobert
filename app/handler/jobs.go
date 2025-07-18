package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/app/worker"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"gorm.io/gorm"
)

type JobCtrl struct {
	Ctrl
	workermu sync.Mutex
	workers  map[string]worker.Worker
}

func NewJobCtrl(store *model.Store, acl *ACL) *JobCtrl {
	return &JobCtrl{
		Ctrl:    BaseCtrl{store, acl},
		workers: make(map[string]worker.Worker),
	}
}

func (ctrl *JobCtrl) Workers() []worker.Worker {
	ctrl.workermu.Lock()
	defer ctrl.workermu.Unlock()

	workers := make([]worker.Worker, 0, len(ctrl.workers))
	for _, worker := range ctrl.workers {
		workers = append(workers, worker)
	}
	return workers
}

func (ctrl *JobCtrl) ListMods(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := ctrl.Store().GetEvidence(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	modules := worker.Supported(obj)
	jobs, err := ctrl.Store().GetJobs(obj.ID)
	if err != nil {
		Err(w, r, err)
		return
	}

	m := fp.ToMap(jobs, func(obj model.Job) string { return obj.Name })
	runs := fp.Apply(modules, func(obj worker.Module) worker.Module {
		return worker.Module{
			Name:        obj.Name,
			Description: obj.Description,
			Status:      m[obj.Name].Status,
			Error:       m[obj.Name].Error,
		}
	})

	Render(w, r, http.StatusOK, views.EvidencesProcess(Env(ctrl, r), runs))
}

func (ctrl *JobCtrl) registerWorker(w http.ResponseWriter, r *http.Request) (string, []string) {
	// create worker id
	workerid := fp.Random(20)
	modules := strings.Split(r.URL.Query().Get("modules"), ",")
	workers, _ := strconv.Atoi(r.URL.Query().Get("workers"))
	log.Printf("worker %q started", workerid)

	// register worker
	ctrl.workermu.Lock()
	ctrl.workers[workerid] = worker.Worker{
		WorkerID:   workerid,
		RemoteAddr: r.RemoteAddr,
		Modules:    modules,
		Workers:    workers,
	}
	ctrl.workermu.Unlock()

	return workerid, modules
}

func (ctrl *JobCtrl) PopJob(w http.ResponseWriter, r *http.Request) {
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

	// register worker
	workerid, modules := ctrl.registerWorker(w, r)

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
			job, kase, evidence, err := ctrl.Store().PopJob(workerid, modules)
			if err != nil && err != gorm.ErrRecordNotFound {
				log.Printf("error fetching job: %v", err)
				goto cleanup
			} else if err == gorm.ErrRecordNotFound {
				continue
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
	err := ctrl.Store().RescheduleWorkerJobs(workerid)
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

func (ctrl *JobCtrl) AckJob(w http.ResponseWriter, r *http.Request) {
	dto := model.Job{}
	if err := Decode(ctrl.Store(), r, &dto, nil); err != nil {
		Err(w, r, err)
		return
	}

	err := ctrl.Store().AckJob(dto.ID, dto.Status, dto.Error)
	if err != nil {
		Err(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ctrl *JobCtrl) PushJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	name := r.FormValue("name")
	err := ctrl.Store().PushJob(model.Job{
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

	ctrl.ListMods(w, r)
}
