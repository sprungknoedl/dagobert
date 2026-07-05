package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerdictSeverity(t *testing.T) {
	tests := []struct {
		v    string
		rank int
		ok   bool
	}{
		{"malicious", 3, true},
		{"suspicious", 2, true},
		{"clean", 1, true},
		{"unknown", 0, true},
		{"Malicious", 3, true},
		{"SUSPICIOUS", 2, true},
		{"Clean", 1, true},
		{"UNKNOWN", 0, true},
		{"harmless", 0, false},
		{"not found", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		rank, ok := VerdictSeverity(tt.v)
		assert.Equal(t, tt.ok, ok, "ok for %q", tt.v)
		if tt.ok {
			assert.Equal(t, tt.rank, rank, "rank for %q", tt.v)
		}
	}
}
