package handler

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/attck"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func (h *Handler) Network(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	env := h.Env(r)
	events, err := h.Store.ListEvents(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	hideIndicators := r.URL.Query().Get("hide") == "indicators"

	nodes := map[string]views.Node{}
	edges := []views.Edge{}
	seen := map[views.Edge]bool{}
	link := func(e views.Edge) {
		if !seen[e] {
			seen[e] = true
			edges = append(edges, e)
		}
	}

	for _, ev := range events {
		for i, x := range ev.Assets {
			nodes[x.ID] = assetNode(env, x)
			for _, y := range ev.Assets[i+1:] {
				link(views.Edge{From: min(x.ID, y.ID), To: max(x.ID, y.ID), Kind: "movement"})
			}

			if hideIndicators {
				continue
			}
			for _, y := range ev.Indicators {
				nodes[y.ID] = indicatorNode(env, y)
				link(views.Edge{From: y.ID, To: x.ID, Kind: "observation"})
			}
		}
	}

	// Sorted, so a reload hands vis-network the same input in the same order —
	// which, with the fixed layout seed in the compiler, redraws the same
	// picture instead of scrambling the arrangement the analyst just read.
	list := fp.ToList(nodes)
	slices.SortFunc(list, func(a, b views.Node) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(edges, func(a, b views.Edge) int {
		return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})

	slog.Debug("rendering network", "nodes", len(list), "edges", len(edges))
	Render(w, r, http.StatusOK, views.VisNetwork(env, list, edges, hideIndicators), nil)
}

func assetNode(env views.Env, x model.Asset) views.Node {
	node := views.Node{
		ID:     x.ID,
		Label:  x.Name,
		Kind:   "asset",
		Type:   x.Type,
		Status: x.Status,
		Tip:    views.Tip{Head: x.Name, Rows: tipRows("Type", x.Type, "Address", x.Addr, "Status", x.Status)},
	}
	if uri, ok := env.Allowed("GET", "/cases/"+x.CaseID+"/assets/"+x.ID); ok {
		node.URL = uri
	}
	return node
}

func indicatorNode(env views.Env, x model.Indicator) views.Node {
	node := views.Node{
		ID:     x.ID,
		Label:  x.Value,
		Kind:   "indicator",
		Type:   x.Type,
		Status: x.Status,
		Tip:    views.Tip{Head: x.Value, Rows: tipRows("Type", x.Type, "TLP", x.TLP, "Status", x.Status)},
	}
	if uri, ok := env.Allowed("GET", "/cases/"+x.CaseID+"/indicators/"+x.ID); ok {
		node.URL = uri
	}
	return node
}

// tipRows pairs up label/value arguments, dropping any row whose value is
// unset — an "Address —" line in a hover card is noise, not information.
func tipRows(pairs ...string) [][2]string {
	rows := [][2]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] != "" {
			rows = append(rows, [2]string{pairs[i], pairs[i+1]})
		}
	}
	return rows
}

func (h *Handler) MitreAttack(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	events, err := h.Store.ListEventTechniques(cid)
	if err != nil {
		Err(w, r, err)
		return
	}

	hide := r.URL.Query().Get("hide") == "on"

	var matrix *attck.Matrix
	matrixKey := r.URL.Query().Get("matrix")
	switch matrixKey {
	case "mobile":
		matrix = h.Mitre.Mobile
	case "ics":
		matrix = h.Mitre.ICS
	case "enterprise":
		fallthrough
	default:
		matrixKey = "enterprise"
		matrix = h.Mitre.Enterprise
	}

	counts := map[string]int{}
	maxCount := 0
	for _, ev := range events {
		for _, tid := range ev.Techniques {
			counts[tid] = counts[tid] + 1
			if counts[tid] > maxCount {
				maxCount = counts[tid]
			}
		}
	}

	// Counted against the unfiltered matrix, and before hide narrows it: a
	// technique the timeline names but this matrix does not carry is not
	// coverage of this matrix, and the readout has to say the same thing
	// whether or not the empty columns are on screen.
	stats := mitreCoverage(events, matrix, counts)

	if hide {
		matrix = matrix.Filter(func(t attck.Technique) bool { return counts[t.ID] > 0 })
	}

	Render(w, r, http.StatusOK, views.VisMitreAttack(h.Env(r), counts, maxCount, matrix, hide, matrixKey, stats), nil)
}

// mitreCoverage reduces the technique counts to the three figures the readout
// carries: how much of the framework the case has touched, and how much of the
// timeline is behind it. A technique listed under two tactics is one technique
// and two touched tactics, which is why both are counted by walking the matrix
// rather than by taking len(counts).
func mitreCoverage(events []model.Event, matrix *attck.Matrix, counts map[string]int) views.MitreCoverage {
	stats := views.MitreCoverage{}
	seen := map[string]bool{}

	for tactic := range matrix.Tactics.Values() {
		touched := false
		for _, t := range tactic.Techniques {
			if counts[t.ID] == 0 {
				continue
			}
			touched = true
			if !seen[t.ID] {
				seen[t.ID] = true
				stats.Techniques++
			}
		}
		if touched {
			stats.Tactics++
		}
	}

	for _, ev := range events {
		if slices.ContainsFunc(ev.Techniques, func(tid string) bool { return seen[tid] }) {
			stats.Events++
		}
	}

	return stats
}
