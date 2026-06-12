package attck

import (
	"slices"
	"testing"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func TestOrderedMap(t *testing.T) {
	om := newFromTechniques([]Technique{
		{ID: "T0003", Name: "Charlie"},
		{ID: "T0001", Name: "Alpha"},
		{ID: "T0002", Name: "Bravo"},
	})

	if om.Len() != 3 {
		t.Errorf("Len() = %d, want 3", om.Len())
	}
	if !om.Has("T0001") || om.Has("T9999") {
		t.Errorf("Has(T0001) = %t, Has(T9999) = %t, want true, false",
			om.Has("T0001"), om.Has("T9999"))
	}

	te, ok := om.Get("T0002")
	if !ok || te.Name != "Bravo" {
		t.Errorf("Get(T0002) = %+v, %t, want Bravo, true", te, ok)
	}
	if _, ok := om.Get("T9999"); ok {
		t.Errorf("Get(T9999) ok = true, want false")
	}

	// Values yields in insertion order, not key order
	got := fp.ApplyS(om.Values(), func(te Technique) string { return te.ID })
	want := []string{"T0003", "T0001", "T0002"}
	if !slices.Equal(got, want) {
		t.Errorf("Values() = %v, want %v", got, want)
	}

	// stopping the iteration early must not panic
	count := 0
	for range om.Values() {
		count++
		break
	}
	if count != 1 {
		t.Errorf("early break yielded %d values, want 1", count)
	}
}

func TestOrderedMapEmpty(t *testing.T) {
	for name, om := range map[string]*OrderedMap[string, Tactic]{
		"nil slice": newFromTactics(nil),
		"empty":     newFromTactics([]Tactic{}),
	} {
		if om.Len() != 0 {
			t.Errorf("%s: Len() = %d, want 0", name, om.Len())
		}
		if om.Has("TA0001") {
			t.Errorf("%s: Has(TA0001) = true, want false", name)
		}
		if got := fp.ApplyS(om.Values(), func(ta Tactic) string { return ta.ID }); len(got) != 0 {
			t.Errorf("%s: Values() = %v, want empty", name, got)
		}
	}
}
