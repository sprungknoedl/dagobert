package handler

import (
	"cmp"
	"net/http"
	"slices"

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
	From string `json:"from"`
	To   string `json:"to"`
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

		for _, x := range ev.Indicators {
			nodes[x.ID] = Node{
				ID:    x.ID,
				Label: x.Value,
				Group: "Indicator" + x.Type,
			}
		}

		for _, x := range ev.Assets {
			nodes[x.ID] = Node{
				ID:    x.ID,
				Label: x.Name,
				Group: "Asset" + x.Type,
			}

			for _, y := range ev.Indicators {
				edges = append(edges, Edge{From: y.ID, To: x.ID})
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

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/vis-network.html", map[string]any{
		"title": "Lateral Movement",
		"nodes": fp.ToList(nodes),
		"edges": edges,
	})
}

func (ctrl VisualsCtrl) Timeline(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	_, err := ctrl.store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	Render(ctrl.store, ctrl.acl, w, r, http.StatusOK, "internal/views/vis-timeline.html", map[string]any{
		"title": "Visual Timeline",
	})
}
