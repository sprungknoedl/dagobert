package attck

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func loadTestKB(t *testing.T) *KB {
	t.Helper()
	kb, err := LoadKB(
		"testdata/enterprise-attack.json",
		"testdata/ics-attack.json",
		"testdata/mobile-attack.json",
	)
	if err != nil {
		t.Fatalf("LoadKB() error = %v", err)
	}
	return kb
}

func techniqueIDs(m *Matrix) []string {
	return fp.ApplyS(m.Techniques.Values(), func(t Technique) string { return t.ID })
}

func TestLoadMatrixTactics(t *testing.T) {
	m, err := LoadMatrix("testdata/enterprise-attack.json")
	if err != nil {
		t.Fatalf("LoadMatrix() error = %v", err)
	}

	// tactics must follow the order of tactic_refs in the x-mitre-matrix
	// object, not document or alphabetical order
	names := fp.ApplyS(m.Tactics.Values(), func(ta Tactic) string { return ta.Name })
	want := []string{"Execution", "Initial Access"}
	if !slices.Equal(names, want) {
		t.Errorf("tactic order = %v, want %v", names, want)
	}

	ta, ok := m.Tactics.Get("TA0001")
	if !ok {
		t.Fatalf("Tactics.Get(TA0001) not found")
	}
	if ta.Name != "Initial Access" ||
		ta.ShortName != "initial-access" ||
		ta.StixID != "x-mitre-tactic--initial-access" ||
		ta.URL != "https://attack.mitre.org/tactics/TA0001" ||
		ta.Description == "" {
		t.Errorf("Tactics.Get(TA0001) = %+v", ta)
	}

	// techniques attached to a tactic are sorted by name
	got := fp.Apply(ta.Techniques, func(te Technique) string { return te.ID })
	if want := []string{"T1002", "T1001"}; !slices.Equal(got, want) {
		t.Errorf("initial-access techniques = %v, want %v", got, want)
	}

	// a technique with multiple kill chain phases appears under each tactic
	ta, _ = m.Tactics.Get("TA0002")
	got = fp.Apply(ta.Techniques, func(te Technique) string { return te.ID })
	if want := []string{"T1002"}; !slices.Equal(got, want) {
		t.Errorf("execution techniques = %v, want %v", got, want)
	}
}

func TestLoadMatrixTechniques(t *testing.T) {
	m, err := LoadMatrix("testdata/enterprise-attack.json")
	if err != nil {
		t.Fatalf("LoadMatrix() error = %v", err)
	}

	// sub-techniques, deprecated and revoked objects are excluded; the
	// remaining techniques are sorted by name (Alpha, Orphan, Zeta)
	got := techniqueIDs(m)
	want := []string{"T1002", "T1003", "T1001"}
	if !slices.Equal(got, want) {
		t.Errorf("techniques = %v, want %v", got, want)
	}

	te, ok := m.Techniques.Get("T1001")
	if !ok {
		t.Fatalf("Techniques.Get(T1001) not found")
	}
	if te.Name != "Zeta Technique" ||
		te.StixID != "attack-pattern--zeta" ||
		// the mitre-attack reference must win even if it is not listed first
		te.URL != "https://attack.mitre.org/techniques/T1001" ||
		te.IsSubTechnique {
		t.Errorf("Techniques.Get(T1001) = %+v", te)
	}
	if want := []string{"initial-access"}; !slices.Equal(te.KillChainPases, want) {
		t.Errorf("T1001 kill chain phases = %v, want %v", te.KillChainPases, want)
	}
}

func TestLoadMatrixErrors(t *testing.T) {
	if _, err := LoadMatrix("testdata/does-not-exist.json"); err == nil {
		t.Errorf("LoadMatrix(missing file) error = nil, want error")
	}

	invalid := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalid, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMatrix(invalid); err == nil {
		t.Errorf("LoadMatrix(invalid json) error = nil, want error")
	}
}

func TestLoadKB(t *testing.T) {
	kb := loadTestKB(t)
	if kb.Enterprise.Techniques.Len() != 3 ||
		kb.ICS.Techniques.Len() != 2 ||
		kb.Mobile.Techniques.Len() != 1 {
		t.Errorf("technique counts = %d/%d/%d, want 3/2/1",
			kb.Enterprise.Techniques.Len(), kb.ICS.Techniques.Len(), kb.Mobile.Techniques.Len())
	}

	// a single unreadable matrix fails the whole knowledge base
	_, err := LoadKB(
		"testdata/enterprise-attack.json",
		"testdata/does-not-exist.json",
		"testdata/mobile-attack.json",
	)
	if err == nil {
		t.Errorf("LoadKB(missing ics) error = nil, want error")
	}
}

func TestKBTechniques(t *testing.T) {
	kb := loadTestKB(t)

	// T1001 exists in enterprise and ICS but must only be yielded once
	got := fp.ApplyS(kb.Techniques(), func(te Technique) string { return te.ID })
	want := []string{"T1002", "T1003", "T1001", "T0801", "T1401"}
	if !slices.Equal(got, want) {
		t.Errorf("KB.Techniques() = %v, want %v", got, want)
	}

	// stopping the iteration early must not panic and yields no extras
	count := 0
	for range kb.Techniques() {
		count++
		break
	}
	if count != 1 {
		t.Errorf("early break yielded %d techniques, want 1", count)
	}
}

func TestKBGetTechnique(t *testing.T) {
	kb := loadTestKB(t)
	for id, name := range map[string]string{
		"T1002": "Alpha Technique",  // enterprise
		"T0801": "ICS Technique",    // ics
		"T1401": "Mobile Technique", // mobile
	} {
		te, err := kb.GetTechnique(id)
		if err != nil {
			t.Errorf("GetTechnique(%s) error = %v", id, err)
		} else if te.Name != name {
			t.Errorf("GetTechnique(%s).Name = %q, want %q", id, te.Name, name)
		}
	}

	if _, err := kb.GetTechnique("T9999"); err == nil {
		t.Errorf("GetTechnique(T9999) error = nil, want error")
	}
}

func TestKBGetTactic(t *testing.T) {
	kb := loadTestKB(t)
	for id, name := range map[string]string{
		"TA0001": "Initial Access",         // enterprise
		"TA0106": "Impair Process Control", // ics
		"TA0041": "Execution",              // mobile
	} {
		ta, err := kb.GetTactic(id)
		if err != nil {
			t.Errorf("GetTactic(%s) error = %v", id, err)
		} else if ta.Name != name {
			t.Errorf("GetTactic(%s).Name = %q, want %q", id, ta.Name, name)
		}
	}

	if _, err := kb.GetTactic("TA9999"); err == nil {
		t.Errorf("GetTactic(TA9999) error = nil, want error")
	}
}

func TestMatrixDimensions(t *testing.T) {
	kb := loadTestKB(t)

	// enterprise: 2 tactics, initial-access holds 2 techniques
	if x, y := kb.Enterprise.Size(); x != 2 || y != 2 {
		t.Errorf("Enterprise.Size() = (%d, %d), want (2, 2)", x, y)
	}
	if x, y := kb.Mobile.Size(); x != 1 || y != 1 {
		t.Errorf("Mobile.Size() = (%d, %d), want (1, 1)", x, y)
	}

	empty := &Matrix{Tactics: newFromTactics(nil), Techniques: newFromTechniques(nil)}
	if x, y := empty.Size(); x != 0 || y != 0 {
		t.Errorf("empty Size() = (%d, %d), want (0, 0)", x, y)
	}
}

func TestMatrixFilter(t *testing.T) {
	kb := loadTestKB(t)
	filtered := kb.Enterprise.Filter(func(te Technique) bool { return te.ID == "T1002" })

	if got := techniqueIDs(filtered); !slices.Equal(got, []string{"T1002"}) {
		t.Errorf("filtered techniques = %v, want [T1002]", got)
	}

	// tactics are kept (even if empty), their technique lists are filtered
	if filtered.DimX() != 2 {
		t.Errorf("filtered DimX() = %d, want 2", filtered.DimX())
	}
	for tactic := range filtered.Tactics.Values() {
		got := fp.Apply(tactic.Techniques, func(te Technique) string { return te.ID })
		if !slices.Equal(got, []string{"T1002"}) {
			t.Errorf("filtered %s techniques = %v, want [T1002]", tactic.ID, got)
		}
	}

	// the original matrix is untouched
	if got := techniqueIDs(kb.Enterprise); len(got) != 3 {
		t.Errorf("original techniques after Filter = %v, want 3 entries", got)
	}

	none := kb.Enterprise.Filter(func(Technique) bool { return false })
	if x, y := none.Size(); x != 2 || y != 0 {
		t.Errorf("Filter(none).Size() = (%d, %d), want (2, 0)", x, y)
	}
}
