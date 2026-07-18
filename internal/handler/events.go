package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/sprungknoedl/dagobert/pkg/timesketch"
	"github.com/sprungknoedl/dagobert/pkg/valid"
)

func (h *Handler) EventList(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	assets, err := h.Store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := h.Store.ListIndicatorsLean(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	comments, err := h.Store.CountComments(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	env := h.Env(r)
	Render(w, r, http.StatusOK, views.EventsMany(env, list, assets, indicators, *h.Mitre, comments), list)
}

func (h *Handler) EventExport(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	list, err := h.Store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	kase := GetCase(h.Store, r)
	filename := fmt.Sprintf("%s - %s - Timeline.csv", time.Now().Format("20060102"), kase.Name)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	cw.Write([]string{"ID", "Time", "Type", "Assets", "Indicators", "Event", "Raw", "Source", "Custom"})
	for _, e := range list {
		cw.Write([]string{
			e.ID,
			e.Time.Format(time.RFC3339),
			e.Type,
			strings.Join(fp.Apply(e.Assets, func(x model.Asset) string { return x.Name }), " "),
			strings.Join(fp.Apply(e.Indicators, func(x model.Indicator) string { return x.Value }), " "),
			e.Event,
			e.Raw,
			e.Source,
			e.Custom.JSON(),
		})
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		slog.Error("failed to write event export csv", "err", err, "raddr", r.RemoteAddr, "case", cid)
	}
}

func (h *Handler) EventImportCSV(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	uri := fmt.Sprintf("/cases/%s/events/", cid)

	if err := h.Store.Transaction(func(tx *model.Store) error {
		return ImportCSV(w, r, uri, 9, func(rec []string) {
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

			var custom model.Custom
			if len(rec) > 8 {
				if err := custom.Scan(rec[8]); err != nil {
					Warn(w, r, err)
					return
				}
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
				Source:     rec[7],
				Custom:     custom,
			}

			if err = tx.SaveEvent(cid, obj, true); err != nil {
				Err(w, r, err)
			}
		})

	}); err != nil {
		// ImportCSV already wrote the HTTP response before Transaction() returns,
		// so a commit failure here can only be surfaced via logging.
		slog.Error("event import transaction failed to commit", "err", err, "case", cid)
	}
}

func (h *Handler) EventImportTimesketch(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	kase, err := h.Store.GetCase(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	if !h.Timesketch.Configured() {
		Warn(w, r, errors.New("timesketch integration is not configured"))
		return
	}
	if kase.SketchID == 0 {
		Warn(w, r, errors.New("case is not linked to a Timesketch sketch"))
		return
	}

	sketch, err := h.Timesketch.GetSketch(r.Context(), kase.SketchID)
	if err != nil {
		Warn(w, r, err)
		return
	}

	events, err := h.Timesketch.ExploreAll(r.Context(), kase.SketchID, "*", timesketch.Filter{
		Size:    1024,
		Order:   "asc",
		Indices: fp.Apply(sketch.Timelines, func(t timesketch.Timeline) int { return t.ID }),
		Chips:   []timesketch.Chip{timesketch.StarredEventsChip},
		Fields:  sketch.Mappings,
	})
	if err != nil {
		Warn(w, r, err)
		return
	}

	if err := saveTimesketchEvents(h.Store, cid, events); err != nil {
		Err(w, r, err)
		return
	}

	uri := fmt.Sprintf("/cases/%s/events/", cid)
	http.Redirect(w, r, uri, http.StatusSeeOther)
}

// saveTimesketchEvents maps starred Timesketch events onto the case timeline
// and saves them in one transaction. The stable "_ts_" ID plus save-without-
// override dedups re-imports: an already imported (and possibly analyst-edited)
// event is kept, not duplicated or clobbered.
func saveTimesketchEvents(store *model.Store, cid string, events []timesketch.Event) error {
	return store.Transaction(func(tx *model.Store) error {
		for _, ev := range events {
			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetIndent("", "  ")
			if err := enc.Encode(ev.Source); err != nil {
				return err
			}

			obj := model.Event{
				ID:     "_ts_" + ev.ID,
				CaseID: cid,
				Type:   "Other",
				Time:   model.Time(ev.Datetime),
				Event:  ev.Message,
				Raw:    buf.String(),
				Source: "Timesketch",
			}

			if err := tx.SaveEvent(cid, obj, false); err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *Handler) EventEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	obj := model.Event{ID: id, CaseID: cid}
	if id != "new" {
		var err error
		obj, err = h.Store.GetEvent(cid, id)
		if errors.Is(err, model.ErrNotFound) {
			NotFound(w, r, err)
			return
		} else if err != nil {
			Err(w, r, err)
			return
		}
	}

	assets, err := h.Store.ListAssets(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	indicators, err := h.Store.ListIndicatorsLean(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(w, r, http.StatusOK, views.EventsOne(h.Env(r), obj, assets, indicators, h.Mitre, valid.ValidationError{}), obj)
}

func (h *Handler) EventSave(w http.ResponseWriter, r *http.Request) {
	dto := model.Event{ID: r.PathValue("id"), CaseID: r.PathValue("cid")}
	tmp := struct {
		Assets     []string
		Indicators []string
	}{} // special case: select-multiple :/
	err := JoinV(
		Decode(h.Store, r, &dto, ValidateEvent),
		Decode(h.Store, r, &tmp, nil))
	if vr, ok := err.(valid.ValidationError); err != nil && ok {
		var ev model.Event
		var err1 error
		if dto.ID != "new" {
			ev, err1 = h.Store.GetEvent(dto.CaseID, dto.ID)
		}
		assets, err2 := h.Store.ListAssets(dto.CaseID)
		indicators, err3 := h.Store.ListIndicatorsLean(dto.CaseID)
		if err := errors.Join(err1, err2, err3); err != nil {
			Err(w, r, err)
			return
		}

		// changes in the form to the selects will be lost, but this is easier than the other way around ...
		dto.Assets = ev.Assets
		dto.Indicators = ev.Indicators
		Render(w, r, http.StatusUnprocessableEntity, views.EventsOne(h.Env(r), dto, assets, indicators, h.Mitre, vr), vr)
		return
	} else if err != nil {
		Warn(w, r, err)
		return
	}

	// NOTE: form-only for now — CollectCustom reads r.PostForm, so a JSON API
	// request yields an empty map and won't carry custom values.
	dto.Custom = CollectCustom(r)

	err = h.Store.Transaction(func(tx *model.Store) error {
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

	RedirectAfterSave(w, r, fmt.Sprintf("/cases/%s/events/", dto.CaseID), dto)
}

func (h *Handler) EventDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cid := r.PathValue("cid")
	if r.URL.Query().Get("confirm") != "yes" && !wantsJSON(r) {
		uri := fmt.Sprintf("/cases/%s/events/%s?confirm=yes", cid, id)
		if err := views.ConfirmDialog(uri).Render(r.Context(), w); err != nil {
			slog.Error("failed to render template", "err", err, "raddr", r.RemoteAddr, "method", r.Method, "url", r.URL)
		}
		return
	}

	err := h.Store.DeleteEvent(cid, id)
	if err != nil {
		Err(w, r, err)
		return
	}

	if wantsJSON(r) {
		w.WriteHeader(http.StatusNoContent)
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
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("get asset by name: %w", err)
		} else if err != nil && errors.Is(err, model.ErrNotFound) {
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
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("get indicator by value: %w", err)
		} else if err != nil && errors.Is(err, model.ErrNotFound) {
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
