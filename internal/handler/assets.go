package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
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
	cw.Write([]string{"ID", "Type", "Name", "IP", "Description", "Compromised", "Analysed"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Type,
			e.Name,
			e.IP,
			e.Description,
			strconv.FormatBool(e.Compromised),
			strconv.FormatBool(e.Analysed),
		})
	}

	cw.Flush()
}

func (ctrl AssetCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := r.URL.RequestURI()
	ImportCSV(ctrl.store, w, r, uri, 7, func(rec []string) {
		compromised, err := strconv.ParseBool(rec[5])
		if err != nil {
			utils.Warn(w, r, err)
			return
		}

		analysed, err := strconv.ParseBool(rec[6])
		if err != nil {
			utils.Warn(w, r, err)
			return
		}

		obj := model.Asset{
			ID:          rec[0],
			CaseID:      cid,
			Type:        rec[1],
			Name:        rec[2],
			IP:          rec[3],
			Description: rec[4],
			Compromised: compromised,
			Analysed:    analysed,
		}

		if err = ctrl.store.SaveAsset(cid, obj); err != nil {
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
