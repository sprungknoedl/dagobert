package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type IndicatorCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewIndicatorCtrl(store *model.Store, acl *ACL) *IndicatorCtrl {
	return &IndicatorCtrl{store, acl}
}

func (ctrl IndicatorCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/indicators-many.html", map[string]any{
		"title": "Indicators",
		"rows":  list,
	})
}

func (ctrl IndicatorCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Indicators.csv", time.Now().Format("20060102"), GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Status", "Type", "Value", "TLP", "Source", "Notes"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Status,
			e.Type,
			e.Value,
			e.TLP,
			e.Source,
			e.Notes,
		})
	}

	cw.Flush()
}

func (ctrl IndicatorCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/indicators/", cid)
	ImportCSV(ctrl.store, ctrl.acl, w, r, uri, 7, func(rec []string) {
		obj := model.Indicator{
			ID:     fp.If(rec[0] == "", random(10), rec[0]),
			Status: rec[1],
			Type:   rec[2],
			Value:  rec[3],
			TLP:    rec[4],
			Source: rec[5],
			Notes:  rec[6],
			CaseID: cid,
		}

		if err := ctrl.store.SaveIndicator(cid, obj); err != nil {
			Err(w, r, err)
			return
		}

		Audit(ctrl.store, r, "indicator:"+obj.ID, "Imported indicator: %s=%v", obj.Type, obj.Value)
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
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/indicators-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl IndicatorCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Indicator{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	if vr := ValidateIndicator(dto); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/indicators-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, random(10), dto.ID)
	if err := ctrl.store.SaveIndicator(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "indicator:"+dto.ID, fp.If(new, "Added indicator: %s=%v", "Updated indicator: %s=%v"), dto.Type, dto.Value)
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/indicators/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl IndicatorCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/indicators/%s?confirm=yes", cid, id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	obj, err := ctrl.store.GetIndicator(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	err = ctrl.store.DeleteIndicator(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "indicator:"+obj.ID, "Deleted indicator: %s=%v", obj.Type, obj.Value)
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/indicators/", cid), http.StatusSeeOther)
}
