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

type AssetCtrl struct {
	store *model.Store
}

func NewAssetCtrl(store *model.Store) *AssetCtrl {
	return &AssetCtrl{store}
}

func (ctrl AssetCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindAssets(cid, search, sort)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, "internal/views/assets-many.html", map[string]any{
		"env":   utils.GetEnv(ctrl.store, r),
		"title": "Assets",
		"rows":  list,
	})
}

func (ctrl AssetCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.FindAssets(cid, "", "")
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Assets.csv", time.Now().Format("20060102"), utils.GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Status", "Type", "Name", "Addr", "Notes"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Status,
			e.Type,
			e.Name,
			e.Addr,
			e.Notes,
		})
	}

	cw.Flush()
}

func (ctrl AssetCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := r.URL.RequestURI()
	ImportCSV(ctrl.store, w, r, uri, 6, func(rec []string) {
		obj := model.Asset{
			ID:     rec[0],
			CaseID: cid,
			Status: rec[1],
			Type:   rec[2],
			Name:   rec[3],
			Addr:   rec[4],
			Notes:  rec[5],
		}

		if err := ctrl.store.SaveAsset(cid, obj); err != nil {
			utils.Err(w, r, err)
		}
	})
}

func (ctrl AssetCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Asset{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetAsset(cid, id)
		if err != nil {
			utils.Err(w, r, err)
			return
		}
	}

	utils.Render(ctrl.store, w, r, "internal/views/assets-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl AssetCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Asset{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := utils.Decode(r, &dto); err != nil {
		utils.Warn(w, r, err)
		return
	}

	if vr := ValidateAsset(dto); !vr.Valid() {
		utils.Render(ctrl.store, w, r, "internal/views/assets-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
	}

	dto.ID = utils.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveAsset(dto.CaseID, dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}

func (ctrl AssetCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/assets/%s?confirm=yes", cid, id)
		utils.Render(ctrl.store, w, r, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
	}

	err := ctrl.store.DeleteAsset(cid, id)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}
