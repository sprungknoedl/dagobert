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

// exprReplacer normalises a marker expression after extraction: typographic
// quotes (raw UTF-8 or numeric character references) that office autocorrect
// substitutes become ASCII quotes, and XML entities are decoded. It runs on
// the extracted expression text only, so document prose keeps its typography.
// strings.Replacer performs a single left-to-right pass in which a replacement
// is never re-examined; that gives "&amp;" the required decode-last semantics
// ("&amp;lt;" becomes "&lt;", not "<").
var exprReplacer = strings.NewReplacer(
	"“", `"`, "”", `"`, "‘", `'`, "’", `'`,
	"&#x201C;", `"`, "&#x201c;", `"`, "&#8220;", `"`,
	"&#x201D;", `"`, "&#x201d;", `"`, "&#8221;", `"`,
	"&#x2018;", `'`, "&#8216;", `'`,
	"&#x2019;", `'`, "&#8217;", `'`,
	"&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&amp;", "&",
)

// buildEmission decides what to emit for a given expression text (after
// stripping {{ }}).
func buildEmission(expr string) string {
	if strings.HasPrefix(expr, "/*") && strings.HasSuffix(expr, "*/") {
		return "{{" + expr + "}}"
	}
	if fields := strings.Fields(expr); len(fields) > 0 {
		switch fields[0] {
		case "range", "end", "if", "else", "with", "define", "block", "template", "break", "continue":
			return "{{ " + expr + " }}"
		}
		// Variable assignments emit no value; piping them through xml would
		// silently store the escaped value in the variable instead.
		if strings.HasPrefix(fields[0], "$") && len(fields) > 1 && (fields[1] == ":=" || fields[1] == "=") {
			return "{{ " + expr + " }}"
		}
	}
	return "{{ " + expr + " | xml }}"
}

// isLegacyMarker detects the pre-walker {{tr range …}} / {{p end }} markers so
// they fail with a migration hint instead of `function "tr" not defined`.
func isLegacyMarker(expr string) bool {
	f := strings.Fields(expr)
	if len(f) < 2 || (f[0] != "tr" && f[0] != "p") {
		return false
	}
	switch f[1] {
	case "range", "end", "if", "else", "with":
		return true
	}
	return false
}

// findClosingBraces returns the byte position just after the closing "}}" in
// raw[from:] that lies outside an XML tag and outside a template string
// literal, or -1 if not found. The "}}" may be split across a tag boundary
// (e.g. "}</t><t>}") — office apps sometimes break a run mid-marker; tags
// between the two "}" bytes are ignored. Non-"}" non-tag bytes reset the
// search so "} foo }" is not treated as "}}". Braces inside a double- or
// backquoted string literal ({{ printf "}}" }}) do not close the marker; a
// string literal that is itself split across runs is not supported.
func findClosingBraces(raw []byte, from int) int {
	inTag := false
	inStr := byte(0) // current string delimiter (" or `), 0 = not in a string
	lastBrace := -1  // position of the most recent lone '}' outside a tag
	for i := from; i < len(raw); i++ {
		b := raw[i]
		switch {
		case inTag:
			if b == '>' {
				inTag = false
			}
		case b == '<':
			inTag = true
		case inStr != 0:
			if b == '\\' && inStr == '"' {
				i++ // skip the escaped character
			} else if b == inStr {
				inStr = 0
			}
			lastBrace = -1
		case b == '"' || b == '`':
			inStr = b
			lastBrace = -1
		case b == '}':
			if lastBrace >= 0 {
				return i + 1 // found "}}"; return position after it
			}
			lastBrace = i
		default:
			lastBrace = -1 // non-"}" content breaks a potential "}}"
		}
	}
	return -1
}

// decodeMarkerExpr strips XML tags from the raw marker bytes (which span one
// or more XML runs with interleaved tags), removes the {{ }} delimiters, trims
// whitespace, and normalises typographic quotes and XML entities in the
// expression text.
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
	return exprReplacer.Replace(s)
}

// reconstructMarkers scans raw XML bytes for {{ }} template markers that may
// be split across multiple text runs by office app formatting, stitches them
// back into single well-formed Go template actions, and returns the modified
// XML together with a slice of markers whose src spans are byte ranges in the
// returned (reconstructed) output — not in the original input.
func reconstructMarkers(raw []byte) ([]byte, []marker, error) {
	var (
		scanStart   = -1 // byte offset of the opening '{' of the marker being scanned
		secondBrace = -1 // byte offset of the second '{' (differs from scanStart+1 for split openers)
		pendingOpen = -1 // trailing lone '{' of the previous text run, may pair with a leading '{'
		lastEnd     = 0  // end of the most recently consumed marker
		markers     []marker
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

		// Tags between text runs are formatting noise; a pending '{' or an
		// open marker scan survives them.
		if _, ok := tok.(xml.CharData); !ok {
			continue
		}
		tokenEnd := int(d.InputOffset())

		// Never re-scan bytes of an already consumed marker (text runs inside
		// a multi-run marker span); markers must not overlap.
		searchStart := max(tokenStart, lastEnd)
		if searchStart >= tokenEnd {
			continue
		}

		// When idle, look for an opening {{ in this token's raw bytes.
		if scanStart == -1 {
			seg := raw[searchStart:tokenEnd]
			if pendingOpen >= 0 && seg[0] == '{' {
				// "{" at the end of the previous run + "{" at the start of
				// this one: an opening "{{" split across two runs.
				scanStart, secondBrace = pendingOpen, searchStart
			} else if i := bytes.Index(seg, []byte("{{")); i >= 0 {
				scanStart = searchStart + i
				secondBrace = scanStart + 1
			}
			pendingOpen = -1
			if scanStart == -1 {
				if seg[len(seg)-1] == '{' {
					pendingOpen = tokenEnd - 1
				}
				continue
			}
		}

		// Process all complete markers whose {{ starts at or before this token.
		// findClosingBraces scans raw bytes forward from the marker, crossing
		// tag boundaries, so a single call can resolve a multi-run marker.
		for scanStart != -1 {
			endPos := findClosingBraces(raw, secondBrace+1)
			if endPos < 0 {
				break // }} not yet seen; wait for more tokens
			}
			expr := decodeMarkerExpr(raw[scanStart:endPos])
			if isLegacyMarker(expr) {
				return nil, nil, fmt.Errorf("legacy {{tr …}}/{{p …}} marker at byte %d: "+
					"dedicated-row/-paragraph markers are no longer needed — write "+
					"{{ range … }} / {{ end }} directly; the surrounding row or "+
					"paragraph is detected automatically", scanStart)
			}
			markers = append(markers, marker{src: span{scanStart, endPos}, text: expr})
			lastEnd = endPos
			scanStart = -1

			if endPos >= tokenEnd {
				break // marker ended past this token; next token continues
			}
			// Check the remainder of this token for another opening {{ or a
			// trailing '{' that may pair with the next run.
			if j := bytes.Index(raw[endPos:tokenEnd], []byte("{{")); j >= 0 {
				scanStart = endPos + j
				secondBrace = scanStart + 1
			} else if raw[tokenEnd-1] == '{' {
				pendingOpen = tokenEnd - 1
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
		if m.src.start < prev {
			return nil, nil, fmt.Errorf("internal: overlapping markers at byte %d", m.src.start)
		}
		out = append(out, raw[prev:m.src.start]...)
		outStart := len(out)
		out = append(out, buildEmission(m.text)...)
		outMarkers = append(outMarkers, marker{src: span{outStart, len(out)}, text: m.text})
		prev = m.src.end
	}
	out = append(out, raw[prev:]...)
	return out, outMarkers, nil
}
