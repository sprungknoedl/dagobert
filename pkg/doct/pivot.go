package doct

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

type node struct {
	tag       string
	openSpan  span
	closeSpan span
	parent    *node
	children  []*node
}

// buildNodeTree parses raw XML and returns a synthetic root node whose children
// are the top-level elements. Byte spans are recorded via Decoder.InputOffset().
func buildNodeTree(raw []byte) (*node, error) {
	root := &node{
		tag:       "#document",
		openSpan:  span{0, 0},
		closeSpan: span{len(raw), len(raw)},
	}
	stack := []*node{root}

	d := xml.NewDecoder(bytes.NewReader(raw))
	d.Strict = false

	for {
		tokenStart := int(d.InputOffset())
		tok, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml decode: %w", err)
		}
		tokenEnd := int(d.InputOffset())

		switch v := tok.(type) {
		case xml.StartElement:
			n := &node{
				tag:      v.Name.Local,
				openSpan: span{tokenStart, tokenEnd},
				parent:   stack[len(stack)-1],
			}
			stack[len(stack)-1].children = append(stack[len(stack)-1].children, n)
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) <= 1 {
				continue
			}
			current := stack[len(stack)-1]
			if tokenStart == tokenEnd {
				// Self-closing element: EndElement consumes no bytes.
				current.closeSpan = current.openSpan
			} else {
				current.closeSpan = span{tokenStart, tokenEnd}
			}
			stack = stack[:len(stack)-1]
		}
	}

	return root, nil
}

// lcaNode returns the deepest node whose span fully contains both posA and posB.
func lcaNode(root *node, posA, posB int) *node {
	current := root
	for {
		found := false
		for _, child := range current.children {
			if child.openSpan.start <= posA && child.closeSpan.end >= posB {
				current = child
				found = true
				break
			}
		}
		if !found {
			return current
		}
	}
}

func markerKeyword(text string) string {
	f := strings.Fields(text)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

func isHoistOpener(kw string) bool {
	switch kw {
	case "range", "if", "with":
		return true
	}
	return false
}

// depthDelta returns the nesting depth change for a marker keyword.
func depthDelta(kw string) int {
	switch kw {
	case "range", "if", "with", "define", "block":
		return +1
	case "end":
		return -1
	}
	return 0 // else, template, values, comments
}

// edit replaces the bytes of src with text. A pure insert has
// src.start == src.end; a pure delete has empty text. seq is the index of the
// originating marker: edits at the same position apply in marker document
// order, which makes adjacent and nested hoists come out in the right order.
type edit struct {
	src  span
	text string
	seq  int
}

// applyEdits rebuilds raw in a single forward pass. Edits must not overlap;
// a violation indicates a hoisting bug and fails loudly instead of corrupting
// the template.
func applyEdits(raw []byte, edits []edit) ([]byte, error) {
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].src.start != edits[j].src.start {
			return edits[i].src.start < edits[j].src.start
		}
		return edits[i].seq < edits[j].seq
	})

	var out []byte
	prev := 0
	for _, e := range edits {
		if e.src.start < prev {
			return nil, fmt.Errorf("internal: overlapping edits at byte %d", e.src.start)
		}
		out = append(out, raw[prev:e.src.start]...)
		out = append(out, e.text...)
		prev = e.src.end
	}
	out = append(out, raw[prev:]...)
	return out, nil
}

// chain is an opener marker with its else markers and matching end, found via
// template nesting rules.
type chain struct {
	opener int // index into markers
	elses  []int
	end    int
}

func findChains(markers []marker) ([]chain, error) {
	var chains []chain
	for i, m := range markers {
		kw := markerKeyword(m.text)
		if !isHoistOpener(kw) {
			continue
		}
		c := chain{opener: i, end: -1}
		depth := 1
		for j := i + 1; j < len(markers); j++ {
			kwj := markerKeyword(markers[j].text)
			if depth == 1 && kwj == "else" {
				c.elses = append(c.elses, j)
			}
			depth += depthDelta(kwj)
			if depth == 0 {
				c.end = j
				break
			}
		}
		if c.end < 0 {
			return nil, fmt.Errorf("unclosed {{ %s }} at byte %d", kw, m.src.start)
		}
		chains = append(chains, c)
	}
	return chains, nil
}

var (
	// rowTags are table rows in OOXML (w:tr) and ODF (table:table-row, used by
	// both Writer tables and Calc sheets). Rows are wrapped instead of sibling-
	// hoisted: repeating a row's cells would change the column count.
	rowTags = map[string]bool{"tr": true, "table-row": true}

	// cellTags guard dissolution: an OOXML table cell must keep at least one
	// paragraph, so the last block child of a cell is never dissolved.
	cellTags = map[string]bool{"tc": true, "table-cell": true, "covered-table-cell": true}
)

// anchorOf returns the child of lca whose span contains pos, or nil when pos
// lies in lca's direct character data — i.e. the marker already sits at lca
// level between its children and needs no hoisting.
func anchorOf(lca *node, pos int) *node {
	for _, c := range lca.children {
		if c.openSpan.start <= pos && pos < c.closeSpan.end {
			return c
		}
	}
	return nil
}

// isStructural reports whether a marker emits no output of its own: control
// flow markers and comments. Anything else (value expressions) is content
// that keeps a container from being dissolved.
func isStructural(text string) bool {
	if strings.HasPrefix(text, "/*") {
		return true
	}
	switch markerKeyword(text) {
	case "range", "if", "with", "else", "end":
		return true
	}
	return false
}

// dissolvable reports whether anchor holds no content besides structural
// markers and whitespace — i.e. it is a dedicated marker container (the old
// {{tr}}/{{p}} authoring pattern) that can be replaced by the markers it holds.
func dissolvable(raw []byte, anchor *node, markers []marker) bool {
	inTag := false
	mi := 0
	for i := anchor.openSpan.start; i < anchor.closeSpan.end; {
		for mi < len(markers) && markers[mi].src.end <= i {
			mi++
		}
		if mi < len(markers) && markers[mi].src.start <= i {
			if !isStructural(markers[mi].text) {
				return false // value markers are real content
			}
			i = markers[mi].src.end
			continue
		}
		b := raw[i]
		switch {
		case inTag:
			if b == '>' {
				inTag = false
			}
		case b == '<':
			inTag = true
		default:
			if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
				return false
			}
		}
		i++
	}
	return true
}

// structuralMarkersIn returns the indices of all structural markers contained
// in anchor, in document order. A dedicated container may hold markers from
// several chains (e.g. "{{ end }}{{ range .Next }}" between two loops); the
// dissolution replaces the container with all of them at once.
func structuralMarkersIn(anchor *node, markers []marker) []int {
	var out []int
	for i, m := range markers {
		if m.src.start >= anchor.openSpan.start && m.src.end <= anchor.closeSpan.end && isStructural(m.text) {
			out = append(out, i)
		}
	}
	return out
}

// hoistPivots moves {{ range }}/{{ if }}/{{ with }} … {{ else }} … {{ end }}
// chains whose markers sit in text deep inside the XML tree to the sibling
// boundaries of their Lowest Common Ancestor, so the correct structural unit
// (table row, paragraph, run) repeats. Containers that hold nothing but
// markers are dissolved entirely, reproducing the old {{tr}}/{{p}} semantics
// without the special syntax. Markers that already sit between elements, or
// whole chains inside a single text node (inline loops), are left untouched.
func hoistPivots(raw []byte, markers []marker) ([]byte, error) {
	if len(markers) == 0 {
		return raw, nil
	}

	root, err := buildNodeTree(raw)
	if err != nil {
		return nil, err
	}
	chains, err := findChains(markers)
	if err != nil {
		return nil, err
	}

	var edits []edit
	handled := make(map[int]bool)     // marker index -> already has edits (or stays put)
	dissolved := make(map[*node]bool) // anchors already replaced by their markers

	// moveMarker hoists marker idx to the sibling boundary of its anchor below
	// lca: before the anchor for openers and else, after it for end. Dedicated
	// marker containers are dissolved instead.
	moveMarker := func(lca *node, idx int, after bool) {
		if handled[idx] {
			return
		}
		m := markers[idx]
		anchor := anchorOf(lca, m.src.start)
		if anchor == nil {
			handled[idx] = true // already at the right structural level
			return
		}

		if dissolvable(raw, anchor, markers) && (!cellTags[lca.tag] || len(lca.children) != 1) {
			if !dissolved[anchor] {
				dissolved[anchor] = true
				group := structuralMarkersIn(anchor, markers)
				var text strings.Builder
				for _, g := range group {
					text.WriteString(buildEmission(markers[g].text))
					handled[g] = true
				}
				edits = append(edits, edit{
					src:  span{anchor.openSpan.start, anchor.closeSpan.end},
					text: text.String(),
					seq:  group[0],
				})
			}
			return
		}

		pos := anchor.openSpan.start
		if after {
			pos = anchor.closeSpan.end
		}
		handled[idx] = true
		edits = append(edits,
			edit{src: m.src, seq: idx},
			edit{src: span{pos, pos}, text: buildEmission(m.text), seq: idx},
		)
	}

	for _, c := range chains {
		opener, end := markers[c.opener], markers[c.end]
		lca := lcaNode(root, opener.src.start, end.src.end)

		if lca == root {
			return nil, fmt.Errorf(
				"{{ %s }}/{{ end }} at byte %d and its {{ end }} share no common ancestor below the document root; check for mismatched or unbalanced markers",
				markerKeyword(opener.text), opener.src.start,
			)
		}

		// Row guard: markers in different cells of one row wrap the row itself.
		if rowTags[lca.tag] {
			if len(c.elses) > 0 {
				return nil, fmt.Errorf(
					"{{ else }} at byte %d cannot be hoisted inside a single table row; place the branches in separate rows",
					markers[c.elses[0]].src.start,
				)
			}
			if !handled[c.opener] {
				handled[c.opener] = true
				edits = append(edits,
					edit{src: opener.src, seq: c.opener},
					edit{src: span{lca.openSpan.start, lca.openSpan.start}, text: buildEmission(opener.text), seq: c.opener},
				)
			}
			if !handled[c.end] {
				handled[c.end] = true
				edits = append(edits,
					edit{src: end.src, seq: c.end},
					edit{src: span{lca.closeSpan.end, lca.closeSpan.end}, text: buildEmission(end.text), seq: c.end},
				)
			}
			continue
		}

		moveMarker(lca, c.opener, false)
		for _, e := range c.elses {
			moveMarker(lca, e, false)
		}
		moveMarker(lca, c.end, true)
	}

	if len(edits) == 0 {
		return raw, nil
	}
	return applyEdits(raw, edits)
}
