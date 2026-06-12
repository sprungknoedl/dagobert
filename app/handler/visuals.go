package handler

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"

	"github.com/sprungknoedl/dagobert/app/auth"
	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/sprungknoedl/dagobert/app/views"
	"github.com/sprungknoedl/dagobert/pkg/attck"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

type VisualsCtrl struct {
	Ctrl
	Mitre *attck.KB
}

func NewVisualsCtrl(store *model.Store, acl *auth.ACL, mitre *attck.KB) *VisualsCtrl {
	return &VisualsCtrl{BaseCtrl{store, acl}, mitre}
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

func (ctrl VisualsCtrl) MitreAttack(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	hide := r.URL.Query().Get("hide") == "on"

	var matrix *attck.Matrix
	switch r.URL.Query().Get("matrix") {
	case "mobile":
		matrix = ctrl.Mitre.Mobile
	case "ics":
		matrix = ctrl.Mitre.ICS
	case "enterprise":
		fallthrough
	default:
		matrix = ctrl.Mitre.Enterprise
	}

	counts := map[string]int{}
	for _, ev := range events {
		for _, tid := range ev.Techniques {
			counts[tid] = counts[tid] + 1
		}
	}

	if hide {
		matrix = matrix.Filter(func(t attck.Technique) bool { return counts[t.ID] > 0 })
	}

	Render(w, r, http.StatusOK, views.VisMitreAttack(Env(ctrl, r), counts, matrix, hide))
}
