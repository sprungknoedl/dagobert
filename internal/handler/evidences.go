package handler

import (
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type EvidenceCtrl struct {
	store *model.Store
}

func NewEvidenceCtrl(store *model.Store) *EvidenceCtrl {
	return &EvidenceCtrl{store}
}

func (ctrl EvidenceCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindEvidences(cid, search, sort)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, http.StatusOK, "internal/views/evidences-many.html", map[string]any{
		"title":        "Evidences",
		"rows":         list,
		"humanizeSize": humanizeSize,
	})
}

func (ctrl EvidenceCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.FindEvidences(cid, "", "")
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Evidences.csv", time.Now().Format("20060102"), utils.GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Type", "Name", "Hash", "Size", "Notes"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Name,
			e.Hash,
			strconv.FormatInt(e.Size, 10),
			e.Notes,
		})
	}

	cw.Flush()
}

func (ctrl EvidenceCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/evidences/", cid)
	ImportCSV(ctrl.store, w, r, uri, 7, func(rec []string) {
		size, err := strconv.ParseInt(rec[4], 10, 64)
		if err != nil {
			utils.Warn(w, r, err)
			return
		}

		obj := model.Evidence{
			ID:     rec[0],
			CaseID: cid,
			Type:   rec[1],
			Name:   rec[2],
			Hash:   rec[3],
			Size:   size, // rec[4]
			Notes:  rec[5],
		}

		err = ctrl.store.SaveEvidence(cid, obj)
		utils.Err(w, r, err)
	})
}

func (ctrl EvidenceCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Evidence{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetEvidence(cid, id)
		if err != nil {
			utils.Err(w, r, err)
			return
		}
	}

	utils.Render(ctrl.store, w, r, http.StatusOK, "internal/views/evidences-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl EvidenceCtrl) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj, err := ctrl.store.GetEvidence(cid, id)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", obj.Name))
	w.WriteHeader(http.StatusOK)
	http.ServeFile(w, r, obj.Location)
}

func (ctrl EvidenceCtrl) Save(w http.ResponseWriter, r *http.Request) {
	// get handle to form file
	fr, fh, err := r.FormFile("File")
	if err != nil && err != http.ErrMissingFile {
		utils.Warn(w, r, err)
		return
	}

	dto := model.Evidence{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := utils.Decode(r, &dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	// default values
	dto.Size = int64(0)
	dto.Hash = ""
	dto.Location = ""
	dto.Name = filepath.Base(dto.Name) // sanitize name
	if vr := ValidateEvidence(dto); !vr.Valid() {
		utils.Render(ctrl.store, w, r, http.StatusUnprocessableEntity, "internal/views/evidences-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	// process file if present
	if fh != nil && fh.Size > 0 {
		// prepare location for evidence storage
		dto.Location = filepath.Join("./files/evidences", filepath.Base(dto.CaseID), dto.Name)
		err = os.MkdirAll(filepath.Dir(dto.Location), 0755)
		if err != nil {
			utils.Err(w, r, err)
			return
		}

		// create file
		fw, err := os.Create(dto.Location)
		if err != nil {
			utils.Err(w, r, err)
			return
		}

		// write and file and simultanously calculate sha1 hash
		hasher := sha1.New()
		mw := io.MultiWriter(fw, hasher)
		_, err = io.Copy(mw, fr)
		if err != nil {
			utils.Err(w, r, err)
			return
		}

		dto.Size = fh.Size
		dto.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	} else if dto.ID != "new" {
		// keep metadata for existing evidences that did not change
		obj, err := ctrl.store.GetEvidence(dto.CaseID, dto.ID)
		if err != nil {
			utils.Err(w, r, err)
			return
		}

		dto.Size = obj.Size
		dto.Hash = obj.Hash
		dto.Location = obj.Location
	}

	dto.ID = utils.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveEvidence(dto.CaseID, dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/evidences/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl EvidenceCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/evidences/%s?confirm=yes", cid, id)
		utils.Render(ctrl.store, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	// TODO: delete file from fs
	err := ctrl.store.DeleteEvidence(cid, id)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/evidences/", cid), http.StatusSeeOther)
}

func humanizeSize(s int) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(math.Log(float64(s)) / math.Log(1000))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(1000, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}
