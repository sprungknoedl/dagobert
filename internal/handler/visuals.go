package handler

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/sprungknoedl/dagobert/internal/fp"
	"github.com/sprungknoedl/dagobert/internal/model"
)

type VisualsCtrl struct {
	store *model.Store
	acl   *ACL
}

type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
}

type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Dashes bool   `json:"dashes"`
}

type DataItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Title   string `json:"title"`
	Start   string `json:"start"`
	Group   string `json:"group"`
}

func NewVisualsCtrl(store *model.Store, acl *ACL) *VisualsCtrl {
	return &VisualsCtrl{store, acl}
}

func (ctrl VisualsCtrl) Network(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := ctrl.store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	nodes := map[string]Node{}
	edges := []Edge{}

	for _, ev := range events {
		if ev.Type == "Legitimate" || ev.Type == "Remediation" {
			continue
		}

		for _, x := range ev.Assets {
			nodes[x.ID] = Node{
				ID:    x.ID,
				Label: x.Name,
				Group: "Asset" + x.Type,
			}

			for _, y := range ev.Indicators {
				nodes[y.ID] = Node{
					ID:    y.ID,
					Label: y.Value,
					Group: "Indicator" + y.Type,
				}

				edges = append(edges, Edge{From: y.ID, To: x.ID, Dashes: true})
			}
		}

		if len(ev.Assets) < 2 {
			continue
		}

		src := ev.Assets[0]
		for _, dst := range ev.Assets[1:] {
			edges = append(edges, Edge{From: src.ID, To: dst.ID})
		}

	}

	slices.SortFunc(edges, func(a, b Edge) int { return cmp.Compare(a.From+a.To, b.From+b.To) })
	edges = slices.Compact(edges)

	slog.Debug("rendering network", "nodes", nodes, "edges", edges)
	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/vis-network.html", map[string]any{
		"title": "Lateral Movement",
		"nodes": fp.ToList(nodes),
		"edges": edges,
	})
}

func (ctrl VisualsCtrl) Timeline(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := ctrl.store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	items := []DataItem{}
	groups := map[string]DataItem{}
	for _, ev := range events {
		if !ev.Flagged {
			continue
		}

		for _, g := range ev.Assets {
			groups[g.Name] = DataItem{
				ID:      g.Name,
				Content: g.Name,
			}

			items = append(items, DataItem{
				ID:      ev.ID + "_" + g.ID,
				Content: ev.Event,
				Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
				Start:   ev.Time.Format(time.RFC3339),
				Group:   g.Name,
			})
		}
		if len(ev.Assets) == 0 {
			groups["Unknown"] = DataItem{
				ID:      "Unknown",
				Content: "Unknown",
			}

			// add without group when no assets are linked to the event
			items = append(items, DataItem{
				ID:      ev.ID + "_Unknown",
				Content: ev.Event,
				Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
				Start:   ev.Time.Format(time.RFC3339),
				Group:   "Unknown",
			})
		}
	}

	slog.Debug("rendering timeline", "items", items, "groups", groups)
	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/vis-timeline.html", map[string]any{
		"title":  "Visual Timeline",
		"items":  items,
		"groups": fp.ToList(groups),
	})
}
