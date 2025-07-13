package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
	"gorm.io/gorm"
)

type EventCtrl struct {
	Ctrl
	ts *timesketch.Client
}

func NewEventCtrl(store *model.Store, acl *ACL, ts *timesketch.Client) *EventCtrl {
	return &EventCtrl{Ctrl: BaseCtrl{store, acl}, ts: ts}
}

func (ctrl EventCtrl) List(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	assets, err := ctrl.Store().ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	env := Env(ctrl, r)
	views.EventsMany(env, list, assets, indicators).Render(r.Context(), w)
}

func (ctrl EventCtrl) Export(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(ctrl.Store(), r)
	filename := fmt.Sprintf("%s - %s - Timeline.csv", time.Now().Format("20060102"), kase.Name)
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

	ctrl.Store().Transaction(func(tx *model.Store) error {
		return ImportCSV(ctrl.Store(), ctrl.ACL(), w, r, uri, 7, func(rec []string) {
			t, err := time.Parse(time.RFC3339, rec[1])
			if err != nil {
				Warn(w, r, err)
				return
			}

			// import assets (creates new one if they don't exist)
			assets, err := getOrCreateAssets(tx, cid, strings.Split(rec[3], " "))
			if err != nil {
				Err(w, r, err)
				return
			}

			// import indicators (creates new one if they don't exist)
			indicators, err := getOrCreateIndicators(tx, cid, strings.Split(rec[4], " "))
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

			if err = ctrl.Store().SaveEvent(cid, obj, true); err != nil {
				Err(w, r, err)
			}
		})

	})
}

func (ctrl EventCtrl) ImportTimesketch(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	kase, err := ctrl.Store().GetCase(cid)
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

		if err = ctrl.Store().SaveEvent(cid, obj, false); err != nil {
			Err(w, r, err)
			return
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
		obj, err = ctrl.Store().GetEvent(cid, id)
		if err != nil {
			Err(w, r, err)
			return
		}
	}

	assets, err := ctrl.Store().ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := ctrl.Store().ListIndicators(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.EventsOne(Env(ctrl, r), obj, assets, indicators, valid.Result{}))
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
	enums, err := ctrl.Store().ListEnums()
	if err != nil {
		Err(w, r, err)
		return
	}

	if vr := ValidateEvent(dto, enums); !vr.Valid() {
		assets, err := ctrl.Store().ListAssets(dto.CaseID)
		if err != nil {
			Err(w, r, err)
			return
		}

		indicators, err := ctrl.Store().ListIndicators(dto.CaseID)
		if err != nil {
			Err(w, r, err)
			return
		}

		Render(w, r, http.StatusUnprocessableEntity, views.EventsOne(Env(ctrl, r), dto, assets, indicators, vr))
		return
	}

	err = ctrl.Store().Transaction(func(tx *model.Store) error {
		var err error
		dto.Assets, err = getOrCreateAssets(tx, dto.CaseID, tmp.Assets)
		if err != nil {
			return err
		}

		dto.Indicators, err = getOrCreateIndicators(tx, dto.CaseID, tmp.Indicators)
		if err != nil {
			return err
		}

		new := dto.ID == "new"
		dto.ID = fp.If(new, fp.Random(10), dto.ID)
		if err = tx.SaveEvent(dto.CaseID, dto, true); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
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
		views.ConfirmDialog(uri).Render(r.Context(), w)
		return
	}

	err := ctrl.Store().DeleteEvent(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/cases/%s/events/", cid), http.StatusSeeOther)
}

func getOrCreateAssets(db *model.Store, cid string, names []string) ([]model.Asset, error) {
	assets := []model.Asset{}
	for _, asset := range names {
		if asset == "" {
			continue
		}

		obj, err := db.GetAssetByName(cid, asset)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("get asset by name: %w", err)
		} else if err != nil && err == gorm.ErrRecordNotFound {
			obj = model.Asset{
				ID:     fp.Random(10),
				CaseID: cid,
				Name:   asset,
				Status: "Under investigation",
				Type:   "Other",
			}
			if err := db.SaveAsset(cid, obj); err != nil {
				return nil, fmt.Errorf("save asset: %w", err)
			}
		}

		assets = append(assets, obj)
	}

	return assets, nil
}

func getOrCreateIndicators(db *model.Store, cid string, values []string) ([]model.Indicator, error) {
	indicators := []model.Indicator{}
	for _, indicator := range values {
		if indicator == "" {
			continue
		}

		obj, err := db.GetIndicatorByValue(cid, indicator)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("get indicator by value: %w", err)
		} else if err != nil && err == gorm.ErrRecordNotFound {
			obj = model.Indicator{
				ID:     fp.Random(10),
				CaseID: cid,
				Value:  indicator,
				Status: "Under investigation",
				Type:   "Other",
				TLP:    "TLP:RED",
			}
			if err := db.SaveIndicator(cid, obj, false); err != nil {
				return nil, fmt.Errorf("save indicator: %w", err)
			}
		}

		indicators = append(indicators, obj)
	}

	return indicators, nil
}
