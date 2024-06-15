package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type IndicatorCtrl struct {
	store *model.Store
}

func NewIndicatorCtrl(store *model.Store) *IndicatorCtrl {
	return &IndicatorCtrl{store}
}

func (ctrl IndicatorCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindIndicators(cid, search, sort)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, "internal/views/indicators-many.html", map[string]any{
		"title": "Indicators",
		"rows":  list,
	})
}

func (ctrl IndicatorCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.FindIndicators(cid, "", "")
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Indicators.csv", time.Now().Format("20060102"), utils.GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Type", "Value", "TLP", "Description", "Source"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Value,
			e.TLP,
			e.Description,
			e.Source,
		})
	}

	cw.Flush()
}

func (ctrl IndicatorCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := r.URL.RequestURI()
	ImportCSV(ctrl.store, w, r, uri, 6, func(rec []string) {
		obj := model.Indicator{
			ID:          rec[0],
			Type:        rec[1],
			Value:       rec[2],
			TLP:         rec[3],
			Description: rec[4],
			Source:      rec[5],
			CaseID:      cid,
		}

		err := ctrl.store.SaveIndicator(cid, obj)
		utils.Err(w, r, err)
	})
}

func (ctrl IndicatorCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Indicator{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetIndicator(cid, id)
		if err != nil {
			utils.Err(w, r, err)
			return
		}
	}

	utils.Render(ctrl.store, w, r, "internal/views/indicators-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl IndicatorCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Indicator{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := utils.Decode(r, &dto); err != nil {
		utils.Warn(w, r, err)
		return
	}

	if vr := ValidateIndicator(dto); !vr.Valid() {
		utils.Render(ctrl.store, w, r, "internal/views/indicators-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.ID = utils.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveIndicator(dto.CaseID, dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}

func (ctrl IndicatorCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/indicators/%s?confirm=yes", cid, id)
		utils.Render(ctrl.store, w, r, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
	}

	err := ctrl.store.DeleteIndicator(cid, id)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}
