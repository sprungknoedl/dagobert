package handler

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type EventCtrl struct {
	store *model.Store
}

func NewEventCtrl(store *model.Store) *EventCtrl {
	return &EventCtrl{
		store: store,
	}
}

func (ctrl EventCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	list, err := ctrl.store.FindEvents(cid, search, sort)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/events-many.html", map[string]any{
		"rows": list,
		"hasTimeGap": func(list []model.Event, i int) string {
			if i > 0 {
				prev := list[i-1].Time
				curr := list[i].Time
				if d := curr.Sub(prev); d.Abs() > 2*24*time.Hour {
					return humanizeDuration(d.Abs())
				}
			}

			return ""
		},
	})
}

func (ctrl EventCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.FindEvents(cid, "", "")
	if err != nil {
		Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Timeline.csv", time.Now().Format("20060102"), GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Time", "Type", "Assets", "Indicators", "Event", "Raw"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Time.Format(time.RFC3339),
			e.Type,
			strings.Join(fp.Apply(e.Assets, func(x model.Asset) string { return x.Name }), " "),
			strings.Join(fp.Apply(e.Indicators, func(x model.Indicator) string { return x.Value }), " "),
			e.Event,
			e.Raw,
		})
	}

	cw.Flush()
}

func (ctrl EventCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/events/", cid)
	ImportCSV(ctrl.store, w, r, uri, 7, func(rec []string) {
		t, err := time.Parse(time.RFC3339, rec[1])
		if err != nil {
			Warn(w, r, err)
			return
		}

		// import assets (creates new one if they don't exist)
		assets := []model.Asset{}
		for _, asset := range strings.Split(rec[3], " ") {
			if asset == "" {
				continue
			}

			obj, err := ctrl.store.GetAssetByName(cid, asset)
			if err != nil && err != sql.ErrNoRows {
				Err(w, r, fmt.Errorf("get asset by name: %w", err))
				return
			} else if err != nil && err == sql.ErrNoRows {
				obj, err = ctrl.store.SaveAsset(cid, model.Asset{
					Name:   asset,
					Status: "Under investigation",
					Type:   "Other",
				})
				if err != nil {
					Err(w, r, fmt.Errorf("save asset: %w", err))
					return
				}
			}

			assets = append(assets, obj)
		}

		// import indicators (creates new one if they don't exist)
		indicators := []model.Indicator{}
		for _, indicator := range strings.Split(rec[4], " ") {
			if indicator == "" {
				continue
			}

			obj, err := ctrl.store.GetIndicatorByValue(cid, indicator)
			if err != nil && err != sql.ErrNoRows {
				Err(w, r, err)
				return
			} else if err != nil && err == sql.ErrNoRows {
				obj, err = ctrl.store.SaveIndicator(cid, model.Indicator{
					Value:  indicator,
					Status: "Under investigation",
					Type:   "Other",
					TLP:    "TLP:RED",
				})
				if err != nil {
					Err(w, r, err)
					return
				}
			}

			indicators = append(indicators, obj)
		}

		obj := model.Event{
			ID:         rec[0],
			CaseID:     cid,
			Time:       t,
			Type:       rec[2],
			Assets:     assets,
			Indicators: indicators,
			Event:      rec[5],
			Raw:        rec[6],
		}

		if err = ctrl.store.SaveEvent(cid, obj); err != nil {
			Err(w, r, err)
		}
	})
}

func (ctrl EventCtrl) Edit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Event{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = ctrl.store.GetEvent(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	assets, err := ctrl.store.FindAssets(cid, "", "")
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := ctrl.store.FindIndicators(cid, "", "")
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, w, r, http.StatusOK, "internal/views/events-one.html", map[string]any{
		"obj":        obj,
		"assets":     assets,
		"indicators": indicators,
		"valid":      valid.Result{},
	})
}

func (ctrl EventCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Event{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	if err := Decode(r, &dto); err != nil {
		Warn(w, r, err)
		return
	}

	// special case: select-multiple :/
	tmp := struct {
		Assets     []string
		Indicators []string
	}{}
	if err := Decode(r, &tmp); err != nil {
		Warn(w, r, err)
		return
	}

	// create any newly specified assets
	for i, elem := range tmp.Assets {
		if strings.HasPrefix(elem, "new:") {
			obj, err := ctrl.store.SaveAsset(dto.CaseID, model.Asset{
				Name:   strings.TrimPrefix(elem, "new:"),
				Status: "Under investigation",
				Type:   "Other",
			})
			if err != nil {
				Err(w, r, err)
				return
			}

			tmp.Assets[i] = obj.ID
		}

	}

	// create any newly specified indicators
	for i, elem := range tmp.Indicators {
		if strings.HasPrefix(elem, "new:") {
			obj, err := ctrl.store.SaveIndicator(dto.CaseID, model.Indicator{
				Value:  strings.TrimPrefix(elem, "new:"),
				Status: "Under investigation",
				Type:   "Other",
				TLP:    "TLP:RED",
			})
			if err != nil {
				Err(w, r, err)
				return
			}

			tmp.Indicators[i] = obj.ID
		}

	}

	dto.Assets = fp.Apply(tmp.Assets, func(id string) model.Asset { return model.Asset{ID: id} })
	dto.Indicators = fp.Apply(tmp.Indicators, func(id string) model.Indicator { return model.Indicator{ID: id} })
	if vr := ValidateEvent(dto); !vr.Valid() {
		assets, err := ctrl.store.FindAssets(dto.CaseID, "", "")
		if err != nil {
			Err(w, r, err)
			return
		}

		indicators, err := ctrl.store.FindIndicators(dto.CaseID, "", "")
		if err != nil {
			Err(w, r, err)
			return
		}

		Render(ctrl.store, w, r, http.StatusUnprocessableEntity, "internal/views/events-one.html", map[string]any{
			"obj":        dto,
			"assets":     assets,
			"indicators": indicators,
			"valid":      vr,
		})
		return
	}

	dto.ID = fp.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveEvent(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/events/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl EventCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/events/%s?confirm=yes", cid, id)
		Render(ctrl.store, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	err := ctrl.store.DeleteEvent(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/events/", cid), http.StatusSeeOther)
}

func humanizeDuration(duration time.Duration) string {
	days := int64(duration.Hours() / 24)
	hours := int64(math.Mod(duration.Hours(), 24))
	minutes := int64(math.Mod(duration.Minutes(), 60))
	seconds := int64(math.Mod(duration.Seconds(), 60))

	chunks := []struct {
		singularName string
		amount       int64
	}{
		{"day", days},
		{"hour", hours},
		{"minute", minutes},
		{"second", seconds},
	}

	parts := []string{}

	for _, chunk := range chunks {
		switch chunk.amount {
		case 0:
			continue
		case 1:
			parts = append(parts, fmt.Sprintf("%d %s", chunk.amount, chunk.singularName))
		default:
			parts = append(parts, fmt.Sprintf("%d %ss", chunk.amount, chunk.singularName))
		}
	}

	return strings.Join(parts, " ")
}
