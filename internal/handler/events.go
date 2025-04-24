package handler

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

type EventCtrl struct {
	store *model.Store
	acl   *ACL
	ts    *timesketch.Client
}

func NewEventCtrl(store *model.Store, acl *ACL, ts *timesketch.Client) *EventCtrl {
	return &EventCtrl{store, acl, ts}
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

func (ctrl EventCtrl) ImportCSV(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/events/", cid)
	ImportCSV(ctrl.store, ctrl.acl, w, r, uri, 7, func(rec []string) {
		t, err := time.Parse(time.RFC3339, rec[1])
		if err != nil {
			Warn(w, r, err)
			return
		}

		// import assets (creates new one if they don't exist)
		assets, err := ctrl.getOrCreateAssets(r, cid, strings.Split(rec[3], " "))
		if err != nil {
			Err(w, r, err)
			return
		}

		// import indicators (creates new one if they don't exist)
		indicators, err := ctrl.getOrCreateIndicators(r, cid, strings.Split(rec[4], " "))
		if err != nil {
			Err(w, r, err)
			return
		}

		obj := model.Event{
			ID:         fp.If(rec[0] == "", fp.Random(10), rec[0]),
			CaseID:     cid,
			Time:       model.Time(t),
			Type:       rec[2],
			Assets:     assets,
			Indicators: indicators,
			Event:      rec[5],
			Raw:        rec[6],
		}

		if err = ctrl.store.SaveEvent(cid, obj, true); err != nil {
			Err(w, r, err)
		} else {
			Audit(ctrl.store, r, "event:"+obj.ID, "Imported event %q", obj.Event)
		}
	})
}

func (ctrl EventCtrl) ImportTimesketch(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	kase, err := ctrl.store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	if ctrl.ts == nil || kase.SketchID == 0 {
		Err(w, r, errors.New("invalid timesketch configuration"))
		return
	}

	sketch, err := ctrl.ts.GetSketch(kase.SketchID)
	if err != nil {
		Err(w, r, err)
		return
	}

	events, err := ctrl.ts.Explore(1, "*", timesketch.Filter{
		Size:    1024,
		Order:   "asc",
		Indices: fp.Apply(sketch.Timelines, func(t timesketch.Timeline) int { return t.ID }),
		Chips:   []timesketch.Chip{timesketch.StarredEventsChip},
		Fields:  sketch.Mappings,
	})
	if err != nil {
		Err(w, r, err)
		return
	}

	for _, ev := range events {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "  ")
		enc.Encode(ev.Source) // FIXME: ignore json errors?

		obj := model.Event{
			ID:     "_ts_" + ev.ID,
			CaseID: cid,
			Type:   "Other",
			Time:   model.Time(ev.Datetime),
			Event:  ev.Message,
			Raw:    buf.String(),
		}

		if err = ctrl.store.SaveEvent(cid, obj, false); err != nil {
			Err(w, r, err)
			return
		} else {
			Audit(ctrl.store, r, "event:"+obj.ID, "Imported event from timesketch %q", obj.Event)
		}
	}

	uri := fmt.Sprintf("/cases/%s/events/", cid)
	http.Redirect(w, r, uri, http.StatusSeeOther)
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

	// TODO: validate before creating assets/indicators

	var err error
	dto.Assets, err = ctrl.getOrCreateAssets(r, dto.CaseID, tmp.Assets)
	if err != nil {
		Err(w, r, err)
		return
	}

	dto.Indicators, err = ctrl.getOrCreateIndicators(r, dto.CaseID, tmp.Indicators)
	if err != nil {
		Err(w, r, err)
		return
	}

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
	dto.ID = fp.If(new, fp.Random(10), dto.ID)
	if err := ctrl.store.SaveEvent(dto.CaseID, dto, true); err != nil {
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

func (ctrl EventCtrl) getOrCreateAssets(r *http.Request, cid string, names []string) ([]model.Asset, error) {
	assets := []model.Asset{}
	for _, asset := range names {
		if asset == "" {
			continue
		}

		obj, err := ctrl.store.GetAssetByName(cid, asset)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("get asset by name: %w", err)
		} else if err != nil && err == sql.ErrNoRows {
			obj = model.Asset{
				ID:     fp.Random(10),
				CaseID: cid,
				Name:   asset,
				Status: "Under investigation",
				Type:   "Other",
			}
			if err := ctrl.store.SaveAsset(cid, obj); err != nil {
				return nil, fmt.Errorf("save asset: %w", err)
			}

			Audit(ctrl.store, r, "asset:"+obj.ID, "Added asset: %q", obj.Name)
		}

		assets = append(assets, obj)
	}

	return assets, nil
}

func (ctrl EventCtrl) getOrCreateIndicators(r *http.Request, cid string, values []string) ([]model.Indicator, error) {
	indicators := []model.Indicator{}
	for _, indicator := range values {
		if indicator == "" {
			continue
		}

		obj, err := ctrl.store.GetIndicatorByValue(cid, indicator)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("get indicator by value: %w", err)
		} else if err != nil && err == sql.ErrNoRows {
			obj := model.Indicator{
				ID:     fp.Random(10),
				CaseID: cid,
				Value:  indicator,
				Status: "Under investigation",
				Type:   "Other",
				TLP:    "TLP:RED",
			}
			if err := ctrl.store.SaveIndicator(cid, obj, false); err != nil {
				return nil, fmt.Errorf("save indicator: %w", err)
			}

			Audit(ctrl.store, r, "indicator:"+obj.ID, "Added indicator: %s=%q", obj.Type, obj.Value)
		}

		indicators = append(indicators, obj)
	}

	return indicators, nil
}
