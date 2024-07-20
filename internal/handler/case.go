package handler

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type CaseCtrl struct {
	store *model.Store
}

func NewCaseCtrl(store *model.Store) *CaseCtrl {
	return &CaseCtrl{store}
}

func (ctrl CaseCtrl) List(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindCases(search, sort)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/cases-many.html", map[string]any{
		"title": "Cases",
		"rows":  list,
	})
}

func (ctrl CaseCtrl) Export(w http.ResponseWriter, r *http.Request) {
	list, err := ctrl.store.FindCases("", "")
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - Cases.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Name", "Severity", "Classification", "Closed", "Outcome", "Summary"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Name,
			e.Severity,
			e.Classification,
			strconv.FormatBool(e.Closed),
			e.Outcome,
			e.Summary,
		})
	}

	cw.Flush()
}

func (ctrl CaseCtrl) Import(w http.ResponseWriter, r *http.Request) {
	uri := "/"
	ImportCSV(ctrl.store, w, r, uri, 7, func(rec []string) {
		closed, err := strconv.ParseBool(cmp.Or(rec[4], "false"))
		if err != nil {
			Warn(w, r, err)
			return
		}

		obj := model.Case{
			ID:             rec[0],
			Name:           rec[1],
			Severity:       rec[2],
			Classification: rec[3],
			Closed:         closed,
			Outcome:        rec[5],
			Summary:        rec[6],
		}

		if err = ctrl.store.SaveCase(obj); err != nil {
			Err(w, r, err)
		}
	})
}

func (ctrl CaseCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	obj := model.Case{ID: cid}
	if cid != "new" {
		var err error
		obj, err = ctrl.store.GetCase(cid)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/cases-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl CaseCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Case{ID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	if vr := ValidateCase(dto); !vr.Valid() {
		Render(ctrl.store, w, r, http.StatusUnprocessableEntity, "internal/views/cases-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.ID = fp.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveCase(dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (ctrl CaseCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s?confirm=yes", cid)
		Render(ctrl.store, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	if err := ctrl.store.DeleteCase(cid); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
