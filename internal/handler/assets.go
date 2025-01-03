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

type AssetCtrl struct {
	store *model.Store
	acl   *ACL
}

func NewAssetCtrl(store *model.Store, acl *ACL) *AssetCtrl {
	return &AssetCtrl{store, acl}
}

func (ctrl AssetCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/assets-many.html", map[string]any{
		"title": "Assets",
		"rows":  list,
	})
}

func (ctrl AssetCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Assets.csv", time.Now().Format("20060102"), GetEnv(ctrl.store, r).ActiveCase.Name)
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
	ImportCSV(ctrl.store, ctrl.acl, w, r, uri, 6, func(rec []string) {
		obj := model.Asset{
			ID:     fp.If(rec[0] == "", random(10), rec[0]),
			CaseID: cid,
			Status: rec[1],
			Type:   rec[2],
			Name:   rec[3],
			Addr:   rec[4],
			Notes:  rec[5],
		}

		if err := ctrl.store.SaveAsset(cid, obj); err != nil {
			Err(w, r, err)
			return
		}

		Audit(ctrl.store, r, "asset:"+obj.ID, "Imported asset %q", obj.Name)
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
			Err(w, r, err)
			return
		}
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/assets-one.html", map[string]any{
		"obj":   obj,
		"valid": valid.Result{},
	})
}

func (ctrl AssetCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Asset{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	if vr := ValidateAsset(dto); !vr.Valid() {
		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/assets-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, random(10), dto.ID)
	if err := ctrl.store.SaveAsset(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "asset:"+dto.ID, fp.If(new, "Added asset %q", "Updated asset %q"), dto.Name)
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/assets/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl AssetCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/assets/%s?confirm=yes", cid, id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	obj, err := ctrl.store.GetAsset(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	err = ctrl.store.DeleteAsset(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "asset:"+obj.ID, "Deleted asset %q", obj.Name)
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/assets/", cid), http.StatusSeeOther)
}
