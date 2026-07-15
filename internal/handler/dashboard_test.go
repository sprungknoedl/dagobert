package handler

import (
	"testing"
	"time"

	"github.com/sprungknoedl/dagobert/internal/model"
)

func date(s string) model.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return model.Date(t)
}

func TestAggregateDashboardCountsTechniqueOncePerCase(t *testing.T) {
	cases := []model.Case{{ID: "1"}}
	techniquesByCase := map[string][]string{
		"1": {"T1566", "T1566", "T1566"}, // 3 events, same technique
	}

	stats := aggregateDashboard(cases, techniquesByCase)
	if got := stats.TechniqueCounts["T1566"]; got != 1 {
		t.Errorf("got %d cases for T1566, want 1", got)
	}
}

func TestCaseInDateScope(t *testing.T) {
	from := date("2026-01-01")
	to := date("2026-06-30")

	tests := []struct {
		name string
		c    model.Case
		from model.Date
		to   model.Date
		want bool
	}{
		{"inside range", model.Case{OpenedAt: date("2026-03-15")}, from, to, true},
		{"before range", model.Case{OpenedAt: date("2025-12-31")}, from, to, false},
		{"after range", model.Case{OpenedAt: date("2026-07-01")}, from, to, false},
		{"on boundary", model.Case{OpenedAt: from}, from, to, true},
		{"missing date, filter set", model.Case{}, from, to, false},
		{"missing date, no filter", model.Case{}, model.Date{}, model.Date{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := caseInDateScope(tt.c, time.Time(tt.from), time.Time(tt.to)); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAggregateDashboardMedianCloseDays(t *testing.T) {
	// odd count: 2, 4, 6 days -> median 4
	odd := []model.Case{
		{ID: "1", Closed: true, OpenedAt: date("2026-01-01"), ClosedAt: date("2026-01-03")},
		{ID: "2", Closed: true, OpenedAt: date("2026-01-01"), ClosedAt: date("2026-01-05")},
		{ID: "3", Closed: true, OpenedAt: date("2026-01-01"), ClosedAt: date("2026-01-07")},
	}
	stats := aggregateDashboard(odd, nil)
	if !stats.HasMedian || stats.MedianCloseDays != 4 {
		t.Errorf("odd: got median %v (has=%v), want 4", stats.MedianCloseDays, stats.HasMedian)
	}

	// even count: 2, 4, 6, 8 days -> median (4+6)/2 = 5
	even := append(odd, model.Case{ID: "4", Closed: true, OpenedAt: date("2026-01-01"), ClosedAt: date("2026-01-09")})
	stats = aggregateDashboard(even, nil)
	if !stats.HasMedian || stats.MedianCloseDays != 5 {
		t.Errorf("even: got median %v (has=%v), want 5", stats.MedianCloseDays, stats.HasMedian)
	}

	// cases missing a date or still open must not affect the median
	withNoise := append(even,
		model.Case{ID: "5", Closed: false, OpenedAt: date("2026-01-01")}, // still open
		model.Case{ID: "6", Closed: true, OpenedAt: date("2026-01-01")},  // missing ClosedAt
		model.Case{ID: "7", Closed: true, ClosedAt: date("2026-01-09")})  // missing OpenedAt
	stats = aggregateDashboard(withNoise, nil)
	if !stats.HasMedian || stats.MedianCloseDays != 5 {
		t.Errorf("with noise: got median %v (has=%v), want 5", stats.MedianCloseDays, stats.HasMedian)
	}

	stats = aggregateDashboard(nil, nil)
	if stats.HasMedian {
		t.Errorf("no closed cases: got HasMedian=true, want false")
	}
}

func TestAggregateDashboardGroupsEmptyClassificationAsUnset(t *testing.T) {
	cases := []model.Case{
		{ID: "1", Classification: "Phishing"},
		{ID: "2", Classification: ""},
		{ID: "3", Classification: "  "},
	}

	stats := aggregateDashboard(cases, nil)

	var unset, phishing int
	for _, row := range stats.Classification {
		switch row.Value {
		case "Unset":
			unset = row.Count
		case "Phishing":
			phishing = row.Count
		}
	}
	if unset != 2 {
		t.Errorf("got %d unset, want 2", unset)
	}
	if phishing != 1 {
		t.Errorf("got %d phishing, want 1", phishing)
	}
}
