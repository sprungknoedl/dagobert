package virustotal

import (
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSupports(t *testing.T) {
	m := &Module{}

	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{"IP", model.Indicator{Type: "IP", TLP: "TLP:GREEN"}, true},
		{"Domain", model.Indicator{Type: "Domain", TLP: "TLP:AMBER"}, true},
		{"Hash", model.Indicator{Type: "Hash", TLP: "TLP:CLEAR"}, true},
		{"URL", model.Indicator{Type: "URL"}, true},
		{"Path rejected", model.Indicator{Type: "Path"}, false},
		{"Service rejected", model.Indicator{Type: "Service"}, false},
		{"Other rejected", model.Indicator{Type: "Other"}, false},
		{"TLP:RED denied", model.Indicator{Type: "IP", TLP: "TLP:RED"}, false},
		{"non-indicator rejected", model.Evidence{Name: "x.evtx"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Supports(tc.obj))
		})
	}
}
