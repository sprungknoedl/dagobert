package attck

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"sort"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

type collection struct {
	Objects []object `json:"objects"`
}

type object struct {
	Type               string      `json:"type"`
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	ExternalReferences []reference `json:"external_references"`

	Revoked        bool     `json:"revoked"`
	Deprecated     bool     `json:"x_mitre_deprecated"`
	ShortName      string   `json:"x_mitre_shortname"`
	IsSubTechnique bool     `json:"x_mitre_is_subtechnique"`
	KillChainPases []phase  `json:"kill_chain_phases"`
	TacticRefs     []string `json:"tactic_refs"`
}

type reference struct {
	ID     string `json:"external_id"`
	URL    string `json:"url"`
	Source string `json:"source_name"`
}

type phase struct {
	Name string `json:"phase_name"`
}

type KB struct {
	Enterprise *Matrix
	ICS        *Matrix
	Mobile     *Matrix
}

type Matrix struct {
	Tactics    *OrderedMap[string, Tactic]
	Techniques *OrderedMap[string, Technique]
}

type Tactics = *OrderedMap[string, Tactic]
type Tactic struct {
	ID          string
	StixID      string
	URL         string
	Name        string
	ShortName   string
	Description string
	Techniques  []Technique
}

type Techniques = *OrderedMap[string, Technique]
type Technique struct {
	ID             string
	StixID         string
	URL            string
	Name           string
	Description    string
	IsSubTechnique bool
	KillChainPases []string
}

func getMitreID(refs []reference) (string, string) {
	for _, ref := range refs {
		if ref.Source == "mitre-attack" {
			return ref.ID, ref.URL
		}
	}
	return "", ""
}

func LoadKB(enterprise string, ics string, mobile string) (*KB, error) {
	e, err1 := LoadMatrix(enterprise)
	i, err2 := LoadMatrix(ics)
	m, err3 := LoadMatrix(mobile)
	if err := errors.Join(err1, err2, err3); err != nil {
		return nil, err
	}

	return &KB{Enterprise: e, ICS: i, Mobile: m}, nil
}

func (kb *KB) Techniques() iter.Seq[Technique] {
	memory := map[string]bool{}
	return func(yield func(t Technique) bool) {
		for t := range kb.Enterprise.Techniques.Values() {
			seen := memory[t.ID]
			memory[t.ID] = true
			if !seen && !yield(t) {
				return
			}
		}

		for t := range kb.ICS.Techniques.Values() {
			seen := memory[t.ID]
			memory[t.ID] = true
			if !seen && !yield(t) {
				return
			}
		}

		for t := range kb.Mobile.Techniques.Values() {
			seen := memory[t.ID]
			memory[t.ID] = true
			if !seen && !yield(t) {
				return
			}
		}
	}
}

func (kb *KB) GetTechnique(id string) (Technique, error) {
	if t, ok := kb.Enterprise.Techniques.Get(id); ok {
		return t, nil
	}
	if t, ok := kb.ICS.Techniques.Get(id); ok {
		return t, nil
	}
	if t, ok := kb.Mobile.Techniques.Get(id); ok {
		return t, nil
	}
	return Technique{}, fmt.Errorf("technique not found: %s", id)
}

func (kb *KB) GetTactic(id string) (Tactic, error) {
	if t, ok := kb.Enterprise.Tactics.Get(id); ok {
		return t, nil
	}
	if t, ok := kb.ICS.Tactics.Get(id); ok {
		return t, nil
	}
	if t, ok := kb.Mobile.Tactics.Get(id); ok {
		return t, nil
	}
	return Tactic{}, fmt.Errorf("tactic not found: %s", id)
}

func LoadMatrix(path string) (*Matrix, error) {
	order := map[string]int{}
	tactics := map[string]Tactic{}
	techniques := []Technique{}

	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	c := collection{}
	err = json.NewDecoder(fh).Decode(&c)
	if err != nil {
		return nil, err
	}

	for _, obj := range c.Objects {
		if obj.Deprecated || obj.Revoked {
			continue
		}

		switch obj.Type {
		case "x-mitre-matrix":
			for i, stixid := range obj.TacticRefs {
				order[stixid] = i
			}

		case "x-mitre-tactic":
			id, url := getMitreID(obj.ExternalReferences)
			tactics[obj.ShortName] = Tactic{
				ID:          id,
				StixID:      obj.ID,
				URL:         url,
				Name:        obj.Name,
				ShortName:   obj.ShortName,
				Description: obj.Description,
			}

		case "attack-pattern":
			if obj.IsSubTechnique {
				continue
			}
			id, url := getMitreID(obj.ExternalReferences)
			techniques = append(techniques, Technique{
				ID:             id,
				StixID:         obj.ID,
				URL:            url,
				Name:           obj.Name,
				Description:    obj.Description,
				IsSubTechnique: obj.IsSubTechnique,
				KillChainPases: fp.Apply(obj.KillChainPases, func(p phase) string { return p.Name }),
			})
		}
	}

	// sort techniques by name
	sort.Slice(techniques, func(i, j int) bool { return techniques[i].Name < techniques[j].Name })
	for _, technique := range techniques {
		for _, phase := range technique.KillChainPases {
			if tactic, ok := tactics[phase]; ok {
				tactic.Techniques = append(tactic.Techniques, technique)
				tactics[phase] = tactic
			}
		}
	}

	// sort tactics by order defined in x-mitre-matrix stix object :/
	list := fp.ToList(tactics)
	sort.Slice(list, func(i, j int) bool { return order[list[i].StixID] < order[list[j].StixID] })

	return &Matrix{
		Tactics:    newFromTactics(list),
		Techniques: newFromTechniques(techniques),
	}, nil
}

func (m *Matrix) Size() (x int, y int) {
	return m.DimX(), m.DimY()
}

func (m *Matrix) DimX() int {
	x := m.Tactics.Len()
	return x
}

func (m *Matrix) DimY() int {
	y := 0
	for tactic := range m.Tactics.Values() {
		if len(tactic.Techniques) > y {
			y = len(tactic.Techniques)
		}
	}
	return y
}

func (m *Matrix) Filter(fn func(Technique) bool) *Matrix {
	tactics := fp.ApplyS(m.Tactics.Values(), func(t Tactic) Tactic {
		return Tactic{
			ID:          t.ID,
			StixID:      t.StixID,
			URL:         t.URL,
			Name:        t.Name,
			ShortName:   t.ShortName,
			Description: t.Description,
			Techniques:  fp.Filter(t.Techniques, fn),
		}
	})
	techniques := fp.FilterS(m.Techniques.Values(), fn)

	return &Matrix{
		Tactics:    newFromTactics(tactics),
		Techniques: newFromTechniques(techniques),
	}
}
