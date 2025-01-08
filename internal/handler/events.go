package handler

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"html/template"
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
	acl   *ACL
}

func NewEventCtrl(store *model.Store, acl *ACL) *EventCtrl {
	return &EventCtrl{store, acl}
}

func (ctrl EventCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	assets, err := ctrl.store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := ctrl.store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/events-many.html", map[string]any{
		"title": "Timeline",
		"rows":  list,
		"hasTimeGap": func(list []model.Event, i int) string {
			if i > 0 {
				prev := time.Time(list[i-1].Time)
				curr := time.Time(list[i].Time)
				if d := curr.Sub(prev); d.Abs() > 2*24*time.Hour {
					return humanizeDuration(d.Abs())
				}
			}

			return ""
		},
		"highlight": func(ev model.Event) template.HTML {
			html := template.HTMLEscapeString(ev.Event)
			// first highlight linked indicators, then any
			for _, x := range ev.Indicators {
				html = strings.ReplaceAll(html, x.Value, "<span class='text-error'>"+template.HTMLEscapeString(x.Value)+"</span>")
			}
			for _, x := range indicators {
				html = strings.ReplaceAll(html, x.Value, "<span class='text-error'>"+template.HTMLEscapeString(x.Value)+"</span>")
			}

			// first highlight linked assets, then any
			for _, x := range ev.Assets {
				html = strings.ReplaceAll(html, x.Name, "<span class='text-success'>"+template.HTMLEscapeString(x.Name)+"</span>")
				if x.Addr != "" {
					html = strings.ReplaceAll(html, x.Addr, "<span class='text-success'>"+template.HTMLEscapeString(x.Addr)+"</span>")
				}
			}
			for _, x := range assets {
				html = strings.ReplaceAll(html, x.Name, "<span class='text-success'>"+template.HTMLEscapeString(x.Name)+"</span>")
				if x.Addr != "" {
					html = strings.ReplaceAll(html, x.Addr, "<span class='text-success'>"+template.HTMLEscapeString(x.Addr)+"</span>")
				}
			}

			return template.HTML(html)
		},
	})
}

func (ctrl EventCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.store.ListEvents(cid)
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
	ImportCSV(ctrl.store, ctrl.acl, w, r, uri, 7, func(rec []string) {
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
				obj = model.Asset{
					ID:     random(10),
					CaseID: cid,
					Name:   asset,
					Status: "Under investigation",
					Type:   "Other",
				}
				if err := ctrl.store.SaveAsset(cid, obj); err != nil {
					Err(w, r, fmt.Errorf("save asset: %w", err))
					return
				}

				Audit(ctrl.store, r, "asset:"+obj.ID, "Added asset (event import) %q", obj.Name)
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
				obj := model.Indicator{
					ID:     random(10),
					CaseID: cid,
					Value:  indicator,
					Status: "Under investigation",
					Type:   "Other",
					TLP:    "TLP:RED",
				}
				if err := ctrl.store.SaveIndicator(cid, obj); err != nil {
					Err(w, r, err)
					return
				}

				Audit(ctrl.store, r, "indicator:"+obj.ID, "Added indicator (event import): %s=%q", obj.Type, obj.Value)
			}

			indicators = append(indicators, obj)
		}

		obj := model.Event{
			ID:         fp.If(rec[0] == "", random(10), rec[0]),
			CaseID:     cid,
			Time:       model.Time(t),
			Type:       rec[2],
			Assets:     assets,
			Indicators: indicators,
			Event:      rec[5],
			Raw:        rec[6],
		}

		if err = ctrl.store.SaveEvent(cid, obj); err != nil {
			Err(w, r, err)
		} else {
			Audit(ctrl.store, r, "event:"+obj.ID, "Imported event %q", obj.Event)
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

	assets, err := ctrl.store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := ctrl.store.ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/events-one.html", map[string]any{
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
			obj := model.Asset{
				ID:     random(10),
				CaseID: dto.CaseID,
				Name:   strings.TrimPrefix(elem, "new:"),
				Status: "Under investigation",
				Type:   "Other",
			}
			if err := ctrl.store.SaveAsset(dto.CaseID, obj); err != nil {
				Err(w, r, err)
				return
			}

			Audit(ctrl.store, r, "asset:"+obj.ID, "Added asset (event save) %q", obj.Name)
			tmp.Assets[i] = obj.ID
		}

	}

	// create any newly specified indicators
	for i, elem := range tmp.Indicators {
		if strings.HasPrefix(elem, "new:") {
			obj := model.Indicator{
				ID:     random(10),
				CaseID: dto.CaseID,
				Value:  strings.TrimPrefix(elem, "new:"),
				Status: "Under investigation",
				Type:   "Other",
				TLP:    "TLP:RED",
			}
			if err := ctrl.store.SaveIndicator(dto.CaseID, obj); err != nil {
				Err(w, r, err)
				return
			}

			Audit(ctrl.store, r, "indicator:"+obj.ID, "Added indicator (event save): %s=%q", obj.Type, obj.Value)
			tmp.Indicators[i] = obj.ID
		}

	}

	dto.Assets = fp.Apply(tmp.Assets, func(id string) model.Asset { return model.Asset{ID: id} })
	dto.Indicators = fp.Apply(tmp.Indicators, func(id string) model.Indicator { return model.Indicator{ID: id} })
	if vr := ValidateEvent(dto); !vr.Valid() {
		assets, err := ctrl.store.ListAssets(dto.CaseID)
		if err != nil {
			Err(w, r, err)
			return
		}

		indicators, err := ctrl.store.ListIndicators(dto.CaseID)
		if err != nil {
			Err(w, r, err)
			return
		}

		Render(ctrl.store, ctrl.acl, w, r, http.StatusUnprocessableEntity, "internal/views/events-one.html", map[string]any{
			"obj":        dto,
			"assets":     assets,
			"indicators": indicators,
			"valid":      vr,
		})
		return
	}

	new := dto.ID == "new"
	dto.ID = fp.If(new, random(10), dto.ID)
	if err := ctrl.store.SaveEvent(dto.CaseID, dto); err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "event:"+dto.ID, fp.If(new, "Added event %q", "Updated event %q"), dto.Event)
	http.Redirect(w, r, fmt.Sprintf("/cases/%s/events/", dto.CaseID), http.StatusSeeOther)
}

func (ctrl EventCtrl) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" {
		uri := fmt.Sprintf("/cases/%s/events/%s?confirm=yes", cid, id)
		Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/utils-confirm.html", map[string]any{
			"dst": uri,
		})
		return
	}

	obj, err := ctrl.store.GetEvent(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	err = ctrl.store.DeleteEvent(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	Audit(ctrl.store, r, "event:"+obj.ID, "Deleted event %q", obj.Event)
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
