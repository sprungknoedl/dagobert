package doct

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// span is a half-open byte range [start, end) in the original XML.
type span struct{ start, end int }

// marker is a template expression reconstructed from one or more source spans.
type marker struct {
	src  span
	text string
}

// preNormalise replaces typographic quotes with ASCII equivalents before
// tokenisation. XML character entities (&lt;, &gt;, &quot;) are intentionally
// NOT replaced here — replacing raw < in text content produces invalid XML
// that breaks the decoder. Those entities are decoded by decodeMarkerExpr
// after the marker span is located in raw bytes.
func preNormalise(raw []byte) []byte {
	raw = bytes.ReplaceAll(raw, []byte("&#x201C;"), []byte(`"`))
	raw = bytes.ReplaceAll(raw, []byte("&#x201D;"), []byte(`"`))
	raw = bytes.ReplaceAll(raw, []byte("&#x2018;"), []byte(`'`))
	raw = bytes.ReplaceAll(raw, []byte("&#x2019;"), []byte(`'`))
	raw = bytes.ReplaceAll(raw, []byte("“"), []byte(`"`))
	raw = bytes.ReplaceAll(raw, []byte("”"), []byte(`"`))
	raw = bytes.ReplaceAll(raw, []byte("‘"), []byte(`'`))
	raw = bytes.ReplaceAll(raw, []byte("’"), []byte(`'`))
	return raw
}

// buildEmission decides what to emit for a given expression text (after
// stripping {{ }}).
func buildEmission(expr string) string {
	if strings.HasPrefix(expr, "/*") && strings.HasSuffix(expr, "*/") {
		return "{{" + expr + "}}"
	}
	if fields := strings.Fields(expr); len(fields) > 0 {
		switch fields[0] {
		case "range", "end", "if", "else", "with", "define", "block", "template":
			return "{{ " + expr + " }}"
		}
	}
	return "{{ " + expr + " | xml }}"
}

// findClosingBraces returns the byte index of the first "}}" in raw[from:]
// that lies outside an XML tag, or -1 if none is found. Because } is never
// XML-entity-encoded, this scan is reliable on unmodified raw bytes.
func findClosingBraces(raw []byte, from int) int {
	inTag := false
	for i := from; i < len(raw)-1; i++ {
		switch {
		case raw[i] == '<':
			inTag = true
		case inTag && raw[i] == '>':
			inTag = false
		case !inTag && raw[i] == '}' && raw[i+1] == '}':
			return i
		}
	}
	return -1
}

// decodeMarkerExpr strips XML tags from the raw marker bytes (which span one
// or more XML runs with interleaved tags), removes the {{ }} delimiters, trims
// whitespace, and decodes XML character entities in the expression text.
func decodeMarkerExpr(raw []byte) string {
	var content []byte
	inTag := false
	for _, b := range raw {
		switch {
		case b == '<':
			inTag = true
		case inTag && b == '>':
			inTag = false
		case !inTag:
			content = append(content, b)
		}
	}
	s := string(content)
	s = strings.TrimPrefix(s, "{{")
	s = strings.TrimSuffix(s, "}}")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}

// reconstructMarkers scans raw XML bytes for {{ }} template markers that may
// be split across multiple text runs by office app formatting, stitches them
// back into single well-formed Go template actions, and returns the modified
// XML together with a slice of markers whose src spans are byte ranges in the
// returned (reconstructed) output — not in the original input.
func reconstructMarkers(raw []byte) ([]byte, []marker, error) {
	raw = preNormalise(raw)

	var (
		scanStart = -1
		markers   []marker
	)

	d := xml.NewDecoder(bytes.NewReader(raw))
	d.Strict = false

	for {
		tokenStart := int(d.InputOffset())
		tok, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("xml decode: %w", err)
		}

		if _, ok := tok.(xml.CharData); !ok {
			continue
		}
		tokenEnd := int(d.InputOffset())

		// When idle, look for an opening {{ in this token's raw bytes.
		if scanStart == -1 {
			i := bytes.Index(raw[tokenStart:tokenEnd], []byte("{{"))
			if i < 0 {
				continue
			}
			scanStart = tokenStart + i
		}

		// Process all complete markers whose {{ starts at or before this token.
		// findClosingBraces scans raw bytes forward from scanStart, crossing
		// tag boundaries, so a single call can resolve a multi-run marker.
		for scanStart != -1 {
			closingPos := findClosingBraces(raw, scanStart+2)
			if closingPos < 0 {
				break // }} not yet seen; wait for more tokens
			}
			endPos := closingPos + 2
			expr := decodeMarkerExpr(raw[scanStart:endPos])
			markers = append(markers, marker{src: span{scanStart, endPos}, text: expr})
			scanStart = -1

			if endPos >= tokenEnd {
				break // marker ended past this token; next token continues
			}
			// Check whether another {{ follows in the remainder of this token.
			j := bytes.Index(raw[endPos:tokenEnd], []byte("{{"))
			if j >= 0 {
				scanStart = endPos + j
			}
		}
	}

	if scanStart != -1 {
		preview := raw[scanStart:]
		if len(preview) > 40 {
			preview = preview[:40]
		}
		return nil, nil, fmt.Errorf("unclosed marker starting at byte %d (near: %q)", scanStart, preview)
	}

	var out []byte
	var outMarkers []marker
	prev := 0
	for _, m := range markers {
		out = append(out, raw[prev:m.src.start]...)
		outStart := len(out)
		emission := buildEmission(m.text)
		out = append(out, emission...)
		outMarkers = append(outMarkers, marker{src: span{outStart, len(out)}, text: m.text})
		prev = m.src.end
	}
	out = append(out, raw[prev:]...)
	return out, outMarkers, nil
}
