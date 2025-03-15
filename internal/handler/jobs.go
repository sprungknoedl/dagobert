package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/worker"
)

var ServerToken = random(20)

type Module struct {
	Name        string
	Description string
	Status      string
	Error       string
}

type JobCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewJobCtrl(store *model.Store, acl *ACL) *JobCtrl {
	return &JobCtrl{store, acl}
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

	// create worker id
	workerid := random(20)
	log.Printf("worker %q started", workerid)

	for {
		select {
		case <-gone:
			log.Println("client disconnected")
			goto cleanup

		case <-t.C:
			job, err := ctrl.store.PopJob(workerid)
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

			err = json.NewEncoder(w).Encode(worker.Job{
				ID:          job.ID,
				WorkerToken: workerid,
				Name:        job.Name,
				Case:        kase,
				Evidence:    evidence,
			})
			if err != nil {
				log.Printf("error encoding job: %v", err)
				goto cleanup
			}

			err = rc.Flush()
			if err != nil {
				log.Printf("error flushing job: %v", err)
				goto cleanup
			}
		}
	}

cleanup:
	log.Printf("worker %q quit", workerid)
	err := ctrl.store.RescheduleWorkerJobs(workerid)
	if err != nil {
		log.Printf("error rescheduling jobs for %q: %v", workerid, err)
	}
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
		ID:          random(10),
		CaseID:      cid,
		EvidenceID:  id,
		Name:        name,
		Status:      "Scheduled",
		ServerToken: ServerToken,
	})
	if err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "evidence:"+obj.ID, "Run extension %q on evidence %q", name, obj.Name)
	ctrl.ListMods(w, r)
}
