package handler

import (
	"cmp"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/sprungknoedl/dagobert/internal/views"
	"github.com/sprungknoedl/dagobert/pkg/attck"
	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	env := h.Env(r)
	from := parseDashboardDate(r.URL.Query().Get("from"))
	to := parseDashboardDate(r.URL.Query().Get("to"))

	hide := r.URL.Query().Get("hide") == "on"

	var matrix *attck.Matrix
	matrixKey := r.URL.Query().Get("matrix")
	switch matrixKey {
	case "mobile":
		matrix = h.Mitre.Mobile
	case "ics":
		matrix = h.Mitre.ICS
	default:
		matrixKey = "enterprise"
		matrix = h.Mitre.Enterprise
	}

	list, err := h.Store.ListCases()
	if err != nil {
		Err(w, r, err)
		return
	}

	cases := fp.Filter(list, func(c model.Case) bool {
		if _, ok := env.Allowed("GET", "/cases/"+c.ID+"/summary/"); !ok {
			return false
		}
		return caseInDateScope(c, from, to)
	})

	techniquesByCase := map[string][]string{}
	for _, c := range cases {
		events, err := h.Store.ListEventTechniques(c.ID)
		if err != nil {
			Err(w, r, err)
			return
		}
		for _, ev := range events {
			techniquesByCase[c.ID] = append(techniquesByCase[c.ID], ev.Techniques...)
		}
	}

	stats := aggregateDashboard(cases, techniquesByCase)

	if hide {
		matrix = matrix.Filter(func(t attck.Technique) bool { return stats.TechniqueCounts[t.ID] > 0 })
	}

	Render(w, r, http.StatusOK, views.Dashboard(env, from, to, matrixKey, matrix, hide, stats), nil)
}

// parseDashboardDate parses a "2006-01-02" query param into a model.Date,
// returning the zero value (treated as "unset") on empty or malformed input.
func parseDashboardDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// caseInDateScope reports whether c falls within the optional [from, to] range,
// scoped by OpenedAt. Cases without OpenedAt are in scope only when no filter
// is set at all, so a fully-missing date never silently disappears from an
// unfiltered dashboard nor silently appears in a filtered one.
func caseInDateScope(c model.Case, from, to time.Time) bool {
	if c.OpenedAt.IsZero() {
		return from.IsZero() && to.IsZero()
	}
	if !from.IsZero() && time.Time(c.OpenedAt).Before(from) {
		return false
	}
	if !to.IsZero() && time.Time(c.OpenedAt).After(to) {
		return false
	}
	return true
}

// aggregateDashboard computes the headline stats, breakdowns, and per-case
// technique counts from plain inputs, so it is unit-testable without
// HTTP/DB. techniquesByCase carries every technique ID observed on every
// event of a case (with duplicates); a technique used by several events in
// the same case still counts once for that case.
func aggregateDashboard(cases []model.Case, techniquesByCase map[string][]string) views.DashboardStats {
	stats := views.DashboardStats{Total: len(cases), TechniqueCounts: map[string]int{}}

	var closeDays []float64
	for _, c := range cases {
		if c.Closed {
			stats.Closed++
		} else {
			stats.Open++
		}
		if c.Closed && !c.OpenedAt.IsZero() && !c.ClosedAt.IsZero() {
			days := time.Time(c.ClosedAt).Sub(time.Time(c.OpenedAt)).Hours() / 24
			closeDays = append(closeDays, days)
		}

		seen := map[string]bool{}
		for _, tid := range techniquesByCase[c.ID] {
			if seen[tid] {
				continue
			}
			seen[tid] = true
			stats.TechniqueCounts[tid]++
			if stats.TechniqueCounts[tid] > stats.MaxTechniqueCount {
				stats.MaxTechniqueCount = stats.TechniqueCounts[tid]
			}
		}
	}

	if len(closeDays) > 0 {
		sorted := slices.Clone(closeDays)
		slices.Sort(sorted)
		mid := len(sorted) / 2
		if len(sorted)%2 == 1 {
			stats.MedianCloseDays = sorted[mid]
		} else {
			stats.MedianCloseDays = (sorted[mid-1] + sorted[mid]) / 2
		}
		stats.HasMedian = true
	}

	stats.Classification = dashboardBreakdown(fp.Apply(cases, func(c model.Case) string { return c.Classification }))
	stats.Severity = dashboardBreakdown(fp.Apply(cases, func(c model.Case) string { return c.Severity }))
	stats.Outcome = dashboardBreakdown(fp.Apply(cases, func(c model.Case) string { return c.Outcome }))

	return stats
}

// dashboardBreakdown groups values into value->count rows, sorted by count
// descending (ties broken alphabetically), with empty values grouped as
// "Unset" and always sorted last.
func dashboardBreakdown(values []string) []views.DashboardBreakdownRow {
	counts := map[string]int{}
	for _, v := range values {
		counts[cmp.Or(strings.TrimSpace(v), "Unset")]++
	}

	rows := []views.DashboardBreakdownRow{}
	unset := 0
	for value, count := range counts {
		if value == "Unset" {
			unset = count
			continue
		}
		rows = append(rows, views.DashboardBreakdownRow{Value: value, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Value < rows[j].Value
	})
	if unset > 0 {
		rows = append(rows, views.DashboardBreakdownRow{Value: "Unset", Count: unset})
	}
	return rows
}
