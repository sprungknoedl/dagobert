package handler

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"
	"time"

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

func (ctrl VisualsCtrl) Timeline(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := ctrl.Store().ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	groupBy := cmp.Or(r.URL.Query().Get("group"), "asset")
	items := []views.DataItem{}
	groups := map[string]views.DataItem{}

	switch groupBy {
	case "category":
		enums, err := ctrl.Store().ListEnums()
		if err != nil {
			Err(w, r, err)
			return
		}

		for _, ev := range events {
			if !ev.Flagged {
				continue
			}

			category := cmp.Or(ev.Type, "Uncategorized")
			groups[category] = views.DataItem{ID: category, Content: category}
			items = append(items, views.DataItem{
				ID:      ev.ID,
				Content: ev.Event,
				Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
				Start:   ev.Time.Format(time.RFC3339),
				Group:   category,
			})
		}

		rankMap := map[string]int{"Uncategorized": len(enums.EventTypes) + 1}
		for i, e := range enums.EventTypes {
			rankMap[e.Name] = i
		}
		getRank := func(name string) int {
			if r, ok := rankMap[name]; ok {
				return r
			}
			return len(enums.EventTypes) + 2
		}
		groupList := fp.ToList(groups)
		slices.SortFunc(groupList, func(a, b views.DataItem) int {
			ra, rb := getRank(a.Content), getRank(b.Content)
			if ra != rb {
				return cmp.Compare(ra, rb)
			}
			return cmp.Compare(a.Content, b.Content)
		})

		slog.Debug("rendering timeline", "items", items, "groups", groupList)
		Render(w, r, http.StatusOK, views.VisTimeLine(Env(ctrl, r), items, groupList, groupBy))

	default: // "asset"
		for _, ev := range events {
			if !ev.Flagged {
				continue
			}

			for _, g := range ev.Assets {
				groups[g.Name] = views.DataItem{ID: g.Name, Content: g.Name}
				items = append(items, views.DataItem{
					ID:      ev.ID + "_" + g.ID,
					Content: ev.Event,
					Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
					Start:   ev.Time.Format(time.RFC3339),
					Group:   g.Name,
				})
			}
			if len(ev.Assets) == 0 {
				groups["Unknown"] = views.DataItem{ID: "Unknown", Content: "Unknown"}
				items = append(items, views.DataItem{
					ID:      ev.ID + "_Unknown",
					Content: ev.Event,
					Title:   ev.Time.Format(time.RFC3339) + " - " + ev.Event,
					Start:   ev.Time.Format(time.RFC3339),
					Group:   "Unknown",
				})
			}
		}

		groupList := fp.ToList(groups)
		slices.SortFunc(groupList, func(a, b views.DataItem) int {
			return cmp.Compare(a.Content, b.Content)
		})

		slog.Debug("rendering timeline", "items", items, "groups", groupList)
		Render(w, r, http.StatusOK, views.VisTimeLine(Env(ctrl, r), items, groupList, groupBy))
	}
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
