package model

import "strings"

// EnrichmentVerdicts is the strict ordered set of recognised verdict tokens,
// highest severity first. Shared between the view helper and enrichment modules
// so the vocabulary can't drift.
var EnrichmentVerdicts = []string{"malicious", "suspicious", "clean", "unknown"}

var verdictRank = map[string]int{
	"malicious":  3,
	"suspicious": 2,
	"clean":      1,
	"unknown":    0,
}

// VerdictSeverity reports the severity rank (malicious=3 … unknown=0) and
// whether v is a recognised verdict. Matching is case-insensitive.
func VerdictSeverity(v string) (rank int, ok bool) {
	rank, ok = verdictRank[strings.ToLower(v)]
	return
}
