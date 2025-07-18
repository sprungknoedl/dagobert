package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type AssetCtrl struct {
	Ctrl
}

func NewAssetCtrl(store *model.Store, acl *ACL) *AssetCtrl {
	return &AssetCtrl{Ctrl: BaseCtrl{store, acl}}
}

func (ctrl AssetCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.AssetsMany(Env(ctrl, r), "Assets", list))
}

func (ctrl AssetCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(ctrl.Store(), r)
	filename := fmt.Sprintf("%s - %s - Assets.csv", time.Now().Format("20060102"), kase.Name)
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
	uri := fmt.Sprintf("/cases/%s/assets/", cid)
	ImportCSV(ctrl.Store(), ctrl.ACL(), w, r, uri, 6, func(rec []string) {
		obj := model.Asset{
			ID:     fp.If(rec[0] == "", fp.Random(10), rec[0]),
			CaseID: cid,
			Status: rec[1],
			Type:   rec[2],
			Name:   rec[3],
			Addr:   rec[4],
			Notes:  rec[5],
		}

		if err := ctrl.Store().SaveAsset(cid, obj); err != nil {
			Err(w, r, err)
			return
		}
	})
}

func (ctrl AssetCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Asset{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.Store().GetAsset(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.AssetsOne(Env(ctrl, r), obj, valid.ValidationError{}))
}

func (ctrl AssetCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Asset{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err := Decode(ctrl.Store(), r, &dto, ValidateAsset)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.AssetsOne(Env(ctrl, r), dto, vr))
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.Store().SaveAsset(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/assets/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl AssetCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/assets/%s?confirm=yes", cid, id)
		views.ConfirmDialog(uri).Render(r.Context(), w)
		return
	}

	err := ctrl.Store().DeleteAsset(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/assets/", cid), http.StatusSeeOther)
}
