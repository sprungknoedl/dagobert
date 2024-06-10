package handler

import (
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/utils"
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
		utils.Err(w, r, err)
		return
	}

	// indicators, err := ctrl.store.FindIndicators(cid, "", "")
	// if err != nil {
	// 	ErrorHandler(w, r, err)
	//  return
	// }

	utils.Render(ctrl.store, w, r, "internal/views/events-many.html", map[string]any{
		"title": "Timeline",
		"rows":  list,
		"hasTimeGap": func(list []model.Event, i int) string {
			if i > 0 {
				prev := list[i-1].Time
				curr := list[i].Time
				if d := curr.Sub(prev); d > 2*24*time.Hour {
					return humanizeDuration(d)
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
		utils.Err(w, r, err)
		return
	}

	filename := fmt.Sprintf("%s - %s - Timeline.csv", time.Now().Format("20060102"), utils.GetEnv(ctrl.store, r).ActiveCase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Time", "Type", "Assets", "Event", "Raw"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Time.Format(time.RFC3339),
			e.Type,
			strings.Join(utils.Apply(e.Assets, func(x model.Asset) string { return x.Name }), " "),
			e.Event,
			e.Raw,
		})
	}

	cw.Flush()
}

func (ctrl EventCtrl) Import(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := r.URL.RequestURI()
	ImportCSV(ctrl.store, w, r, uri, 6, func(rec []string) {
		t, err := time.Parse(time.RFC3339, rec[1])
		if err != nil {
			utils.Warn(w, r, err)
			return
		}

		obj := model.Event{
			ID:     rec[0],
			CaseID: cid,
			Time:   t,
			Type:   rec[2],
			// TODO: import asset links
			// Assets: rec[3]
			Event: rec[4],
			Raw:   rec[5],
		}

		if err = ctrl.store.SaveEvent(cid, obj); err != nil {
			utils.Err(w, r, err)
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
			utils.Err(w, r, err)
			return
		}
	}

	assets, err := ctrl.store.FindAssets(cid, "", "")
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Render(ctrl.store, w, r, "internal/views/events-one.html", map[string]any{
		"obj":    obj,
		"assets": assets,
		"valid":  valid.Result{},
	})
}

func (ctrl EventCtrl) Save(w http.ResponseWriter, r *http.Request) {
	dto := model.Event{}
	if err := utils.Decode(r, &dto); err != nil {
		utils.Warn(w, r, err)
		return
	}

	// special case: select-multiple :/
	tmp := struct{ Assets []string }{}
	if err := utils.Decode(r, &tmp); err != nil {
		utils.Warn(w, r, err)
		return
	}

	dto.Assets = utils.Apply(tmp.Assets, func(id string) model.Asset { return model.Asset{ID: id} })
	if vr := ValidateEvent(dto); !vr.Valid() {
		// assets, err := ctrl.store.FindAssets(cid, "", "")
		// if err != nil {
		// 	ErrorHandler(w, r, err)
		//  return
		// }
		// names := apply(assets, func(x model.Asset) string { return x.Name })
		utils.Render(ctrl.store, w, r, "internal/views/events-one.html", map[string]any{
			"obj":   dto,
			"valid": vr,
		})
		return
	}

	dto.ID = utils.If(dto.ID == "new", "", dto.ID)
	if err := ctrl.store.SaveEvent(dto.CaseID, dto); err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
}

func (ctrl EventCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/events/%s?confirm=yes", cid, id)
		utils.Render(ctrl.store, w, r, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	err := ctrl.store.DeleteEvent(cid, id)
	if err != nil {
		utils.Err(w, r, err)
		return
	}

	utils.Refresh(w, r)
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
