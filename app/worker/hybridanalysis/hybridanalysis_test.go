package hybridanalysis

import (
	"testing"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/stretchr/testify/assert"
)

func TestSupports(t *testing.T) {
	m := &Module{}

	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{"Hash passes", model.Indicator{Type: "Hash", TLP: "TLP:GREEN"}, true},
		{"Hash CLEAR passes", model.Indicator{Type: "Hash", TLP: "TLP:CLEAR"}, true},
		{"IP rejected", model.Indicator{Type: "IP", TLP: "TLP:GREEN"}, false},
		{"Domain rejected", model.Indicator{Type: "Domain", TLP: "TLP:GREEN"}, false},
		{"URL rejected", model.Indicator{Type: "URL", TLP: "TLP:GREEN"}, false},
		{"Path rejected", model.Indicator{Type: "Path"}, false},
		{"Service rejected", model.Indicator{Type: "Service"}, false},
		{"Other rejected", model.Indicator{Type: "Other"}, false},
		{"TLP:RED denied", model.Indicator{Type: "Hash", TLP: "TLP:RED"}, false},
		{"non-indicator rejected", model.Evidence{Name: "x.evtx"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Supports(tc.obj))
		})
	}
}
