package handler

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

type VisualsCtrl struct {
	Ctrl
}

func NewVisualsCtrl(store *model.Store, acl *ACL) *VisualsCtrl {
	return &VisualsCtrl{BaseCtrl{store, acl}}
}

func (ctrl VisualsCtrl) Network(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	nodes := map[string]views.Node{}
	edges := []views.Edge{}

	for _, ev := range events {
		if ev.Type == "Legitimate" || ev.Type == "Remediation" {
			continue
		}

		for _, x := range ev.Assets {
			nodes[x.ID] = views.Node{
				ID:    x.ID,
				Label: x.Name,
				Group: "Asset" + x.Type,
			}

			for _, y := range ev.Indicators {
				nodes[y.ID] = views.Node{
					ID:    y.ID,
					Label: y.Value,
					Group: "Indicator" + y.Type,
				}

				edges = append(edges, views.Edge{From: y.ID, To: x.ID, Dashes: true})
			}
		}

		if len(ev.Assets) < 2 {
			continue
		}

		src := ev.Assets[0]
		for _, dst := range ev.Assets[1:] {
			edges = append(edges, views.Edge{From: src.ID, To: dst.ID})
		}

	}

	slices.SortFunc(edges, func(a, b views.Edge) int { return cmp.Compare(a.From+a.To, b.From+b.To) })
	edges = slices.Compact(edges)

	slog.Debug("rendering network", "nodes", nodes, "edges", edges)
	Render(w, r, http.StatusOK, views.VisNetwork(Env(ctrl, r), fp.ToList(nodes), edges))
}

func (ctrl VisualsCtrl) Timeline(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	items := []views.DataItem{}
	groups := map[string]views.DataItem{}
	for _, ev := range events {
		if !ev.Flagged {
			continue
		}

		for _, g := range ev.Assets {
			groups[g.Name] = views.DataItem{
				ID:      g.Name,
				Content: g.Name,
			}

			items = append(items, views.DataItem{
				ID:      ev.ID + "_" + g.ID,
				Content: ev.Event,
				Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
				Start:   ev.Time.Format(time.RFC3339),
				Group:   g.Name,
			})
		}
		if len(ev.Assets) == 0 {
			groups["Unknown"] = views.DataItem{
				ID:      "Unknown",
				Content: "Unknown",
			}

			// add without group when no assets are linked to the event
			items = append(items, views.DataItem{
				ID:      ev.ID + "_Unknown",
				Content: ev.Event,
				Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
				Start:   ev.Time.Format(time.RFC3339),
				Group:   "Unknown",
			})
		}
	}

	slog.Debug("rendering timeline", "items", items, "groups", groups)
	Render(w, r, http.StatusOK, views.VisTimeLine(Env(ctrl, r), items, fp.ToList(groups)))
}
