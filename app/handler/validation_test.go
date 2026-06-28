package handler

import (
	"testing"

	"github.com/sprungknoedl/dagobert/app/model"
)

func TestValidateMalwareHash(t *testing.T) {
	tests := []struct {
		hash    string
		invalid bool
	}{
		{"abc123", false},
		{"DEADBEEF0123456789", false},
		{"../../etc/passwd", true},
		{"a/b", true},
		{"a.zip", true},
		{`a\b`, true},
		{"", true}, // empty is Missing, still not accepted
	}
	for _, tt := range tests {
		dto := &model.Malware{Hash: tt.hash, Path: "x", Status: "Clean", Asset: model.Asset{ID: "a1"}}
		vr := ValidateMalware(dto, model.Enums{})
		c, flagged := vr["Hash"]
		if flagged != tt.invalid {
			t.Errorf("hash %q: got flagged=%v, want %v (%+v)", tt.hash, flagged, tt.invalid, c)
		}
	}
}
