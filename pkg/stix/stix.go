// Package stix builds minimal STIX 2.1 bundles of indicator objects.
package stix

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Bundle is a STIX 2.1 bundle envelope.
type Bundle struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Objects []Indicator `json:"objects"`
}

// Indicator is a STIX 2.1 indicator SDO.
type Indicator struct {
	Type        string `json:"type"`
	SpecVersion string `json:"spec_version"`
	ID          string `json:"id"`
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	Pattern     string `json:"pattern"`
	PatternType string `json:"pattern_type"`
	ValidFrom   string `json:"valid_from"`
}

// NewBundle returns an empty STIX 2.1 bundle with a generated id.
func NewBundle() *Bundle {
	return &Bundle{
		ID:   "bundle--" + uuid.NewString(),
		Type: "bundle",
	}
}

// AddIndicator appends an indicator SDO carrying the given STIX pattern, filling
// in every required STIX 2.1 property (id, created, modified, spec_version,
// valid_from) so the resulting object always validates.
func (b *Bundle) AddIndicator(pattern string, now time.Time) {
	ts := Timestamp(now)
	b.Objects = append(b.Objects, Indicator{
		Type:        "indicator",
		SpecVersion: "2.1",
		ID:          "indicator--" + uuid.NewString(),
		Created:     ts,
		Modified:    ts,
		Pattern:     pattern,
		PatternType: "stix",
		ValidFrom:   ts,
	})
}

// Timestamp formats t as a STIX 2.1 timestamp: RFC 3339 in UTC with millisecond
// precision and a trailing "Z".
func Timestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// QuoteLiteral escapes a string for use inside a STIX 2.1 pattern string
// literal. Backslashes and single quotes are the only characters that must be
// escaped.
func QuoteLiteral(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}
