package handler

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) AssetList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	comments, err := h.Store.CountComments(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.AssetsMany(h.Env(r), "Assets", list, comments), list)
}

func (h *Handler) AssetExport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Assets.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Status", "Type", "Name", "Addr", "Notes", "Custom"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Status,
			e.Type,
			e.Name,
			e.Addr,
			e.Notes,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
}

func (h *Handler) AssetImport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/assets/", cid)
	if err := h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(tx, h.ACL, w, r, uri, 7, func(rec []string) {
			var custom model.Custom
			if len(rec) > 6 {
				if err := custom.Scan(rec[6]); err != nil {
					Warn(w, r, err)
					return
				}
			}

			obj := model.Asset{
				ID:     fp.If(rec[0] == "", fp.Random(10), rec[0]),
				CaseID: cid,
				Status: rec[1],
				Type:   rec[2],
				Name:   rec[3],
				Addr:   rec[4],
				Notes:  rec[5],
				Custom: custom,
			}

			if err := tx.SaveAsset(cid, obj); err != nil {
				Err(w, r, err)
				return
			}
		})
	}); err != nil {
		// ImportCSV already wrote the HTTP response before Transaction() returns,
		// so a commit failure here can only be surfaced via logging.
		slog.Error("asset import transaction failed to commit", "err", err, "case", cid)
	}
}

func (h *Handler) AssetEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Asset{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = h.Store.GetAsset(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	Render(w, r, http.StatusOK, views.AssetsOne(h.Env(r), obj, valid.ValidationError{}), obj)
}

func (h *Handler) AssetSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Asset{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	err := Decode(h.Store, r, &dto, ValidateAsset)
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		Render(w, r, http.StatusUnprocessableEntity, views.AssetsOne(h.Env(r), dto, vr), vr)
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// NOTE: form-only for now — CollectCustom reads r.PostForm, so a JSON API
	// request yields an empty map and won't carry custom values.
	dto.Custom = CollectCustom(r)

	new := dto.ID == "new"
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := h.Store.SaveAsset(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/assets/", dto.CaseID), dto)
}

func (h *Handler) AssetDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/assets/%s?confirm=yes", cid, id)
		if err := views.ConfirmDialog(uri).Render(r.Context(), w); err != nil {
			slog.Error("failed to render template", "err", err, "raddr", r.RemoteAddr, "method", r.Method, "url", r.URL)
		}
		return
	}

	err := h.Store.DeleteAsset(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/assets/", cid), http.StatusSeeOther)
}
