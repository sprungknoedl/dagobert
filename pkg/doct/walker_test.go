package doct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeMarkerExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Tags interleaved by split runs are stripped, {{ }} and whitespace
		// trimmed.
		{name: "plain", input: `{{ .Name }}`, want: `.Name`},
		{name: "cross-run", input: `{{ .Na</t><t>me }}`, want: `.Name`},

		// Typographic quotes that office autocorrect injects inside markers
		// become ASCII quotes — as UTF-8 bytes or numeric character references.
		{name: "UTF-8 double quotes", input: "{{ .Foo “bar” }}", want: `.Foo "bar"`},
		{name: "UTF-8 single quotes", input: "{{ .Foo ‘bar’ }}", want: `.Foo 'bar'`},
		{name: "hex refs", input: `{{ .Foo &#x201C;bar&#x201D; }}`, want: `.Foo "bar"`},
		{name: "decimal refs", input: `{{ .Foo &#8220;bar&#8221; }}`, want: `.Foo "bar"`},

		// XML entities are decoded to the operators the author intended.
		{name: "&lt; &gt;", input: `{{ .Count &lt; 5 &gt; .X }}`, want: `.Count < 5 > .X`},
		{name: "&quot;", input: `{{ eq .X &quot;y&quot; }}`, want: `eq .X "y"`},
		{name: "&apos;", input: `{{ .S &apos;x&apos; }}`, want: `.S 'x'`},

		// &amp; decodes one level only: the single-pass replacer never
		// re-examines a replacement, so double-encoded forms stay entities.
		{name: "&amp; decoded last", input: `{{ .A &amp;lt; .B }}`, want: `.A &lt; .B`},
		{name: "&amp; alone", input: `{{ .A &amp; .B }}`, want: `.A & .B`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, decodeMarkerExpr([]byte(tt.input)))
		})
	}
}

func TestBuildEmission(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		// Value expressions get the | xml pipe so data cannot inject raw XML
		{name: "field access", expr: ".Name", want: "{{ .Name | xml }}"},
		{name: "method call", expr: ".Events | len", want: "{{ .Events | len | xml }}"},
		{name: "bare variable emits a value", expr: "$x", want: "{{ $x | xml }}"},

		// Control words are structural; Go template interprets them, not the xml pipe
		{name: "range", expr: "range .Events", want: "{{ range .Events }}"},
		{name: "end", expr: "end", want: "{{ end }}"},
		{name: "if", expr: "if .Cond", want: "{{ if .Cond }}"},
		{name: "else", expr: "else", want: "{{ else }}"},
		{name: "with", expr: "with .X", want: "{{ with .X }}"},
		{name: "define", expr: `define "name"`, want: `{{ define "name" }}`},
		{name: "block", expr: `block "header" .`, want: `{{ block "header" . }}`},
		{name: "template", expr: `template "footer" .`, want: `{{ template "footer" . }}`},
		{name: "break", expr: "break", want: "{{ break }}"},
		{name: "continue", expr: "continue", want: "{{ continue }}"},

		// Assignments emit nothing; piping them would escape the stored value
		{name: "declaration", expr: "$x := .Y", want: "{{ $x := .Y }}"},
		{name: "assignment", expr: "$x = .Y", want: "{{ $x = .Y }}"},

		// A word that merely starts like a control keyword is a value expression
		{name: "ranger is not a control word", expr: "ranger", want: "{{ ranger | xml }}"},
		{name: "endif is not a control word", expr: "endif", want: "{{ endif | xml }}"},

		// Comments pass through as-is; the /* */ delimiters are the discriminator
		{name: "comment", expr: "/* a note */", want: "{{/* a note */}}"},
		{name: "comment without spaces", expr: "/*nospace*/", want: "{{/*nospace*/}}"},
		// A comment that happens to start with a control keyword stays a comment
		{name: "comment containing range", expr: "/* range .X */", want: "{{/* range .X */}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEmission(tt.expr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReconstructMarkers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string // substring the error must contain; empty = no error
	}{
		{
			name:  "single-run marker",
			input: `<t>{{ .Name }}</t>`,
			want:  `<t>{{ .Name | xml }}</t>`,
		},
		{
			// Office apps split runs at arbitrary boundaries; the cross-run
			// span including the intermediate tags must collapse into one action.
			name:  "split marker across two runs",
			input: `<t>{{ .Na</t><t>me }}</t>`,
			want:  `<t>{{ .Name | xml }}</t>`,
		},
		{
			name:  "control word unchanged",
			input: `<t>{{ range .Events }}</t>`,
			want:  `<t>{{ range .Events }}</t>`,
		},
		{
			name:  "comment unchanged",
			input: `<t>{{/* note */}}</t>`,
			want:  `<t>{{/* note */}}</t>`,
		},
		{
			// &#x201C; and &#x201D; are XML character references for typographic
			// double quotes that autocorrect injects; they must become plain ASCII.
			name:  "smart quotes normalised",
			input: `<t>{{ .Foo &#x201C;bar&#x201D; }}</t>`,
			want:  `<t>{{ .Foo "bar" | xml }}</t>`,
		},
		{
			// A marker without a closing }} means the template is broken.
			name:    "unclosed marker",
			input:   `<t>{{ .Foo</t>`,
			wantErr: "unclosed marker",
		},
		{
			// Surrounding XML that contains no markers must be returned unchanged.
			name:  "no markers",
			input: `<t>plain text</t>`,
			want:  `<t>plain text</t>`,
		},
		{
			// Typography in document prose must survive — normalisation only
			// applies to marker expressions, not to report text.
			name:  "prose typography untouched",
			input: "<t>“quoted” prose and Tom’s notes</t>",
			want:  "<t>“quoted” prose and Tom’s notes</t>",
		},
		{
			// Both markers must be replaced; the literal XML between them is kept.
			name:  "two markers in separate elements",
			input: `<t>{{ .First }}</t><t>{{ .Last }}</t>`,
			want:  `<t>{{ .First | xml }}</t><t>{{ .Last | xml }}</t>`,
		},
		{
			// Two complete markers within a single CharData token.
			name:  "two markers in the same run",
			input: `<t>{{ .First }} {{ .Last }}</t>`,
			want:  `<t>{{ .First | xml }} {{ .Last | xml }}</t>`,
		},
		{
			// A marker that spans three runs instead of two.
			name:  "marker split across three runs",
			input: `<t>{{</t><t> ran</t><t>ge .X }}</t>`,
			want:  `<t>{{ range .X }}</t>`,
		},
		{
			// UTF-8 curly quotes stored as raw bytes (not XML entity references).
			name:  "UTF-8 typographic quotes normalised",
			input: "<t>{{ .Foo “bar” }}</t>",
			want:  `<t>{{ .Foo "bar" | xml }}</t>`,
		},
		{
			// end has no arguments; it must not acquire | xml.
			name:  "standalone end control word",
			input: `<t>{{ end }}</t>`,
			want:  `<t>{{ end }}</t>`,
		},
		{
			// Office apps sometimes split "}}" across two runs so the raw bytes
			// contain "}</t><t>}" with no consecutive "}}" anywhere.
			name:  "split closing braces across runs",
			input: `<t>{{ .Event}</t><t>}{{ end }}</t>`,
			want:  `<t>{{ .Event | xml }}{{ end }}</t>`,
		},
		{
			// The opening "{{" can be split across runs as well: a trailing
			// lone "{" pairs with a leading "{" in the next run.
			name:  "split opening braces across runs",
			input: `<t>{</t><t>{ .X }}</t>`,
			want:  `<t>{{ .X | xml }}</t>`,
		},
		{
			// Three-way split: "{" / "{" / expression.
			name:  "opening braces in two separate runs",
			input: `<t>{</t><t>{</t><t> .X }}</t>`,
			want:  `<t>{{ .X | xml }}</t>`,
		},
		{
			// A lone trailing "{" that is NOT followed by a "{" is literal text.
			name:  "trailing brace is not an opener",
			input: `<t>a {</t><t>b</t>`,
			want:  `<t>a {</t><t>b</t>`,
		},
		{
			// "}}" inside a string literal must not close the marker.
			name:  "closing braces inside string literal",
			input: `<t>{{ printf "}}" }}</t>`,
			want:  `<t>{{ printf "}}" | xml }}</t>`,
		},
		{
			// "{{" inside a string literal of a multi-run marker: the text run
			// containing it lies inside the consumed span and must not start a
			// second, overlapping marker.
			name:  "no overlapping marker from consumed span",
			input: `<t>{{ .A "</t><t>{{" }}</t>`,
			want:  `<t>{{ .A "{{" | xml }}</t>`,
		},

		// --- regression: XML entities in document content and markers ---

		{
			name:  "&lt; in document content preserved",
			input: `<t>Price &lt; 100</t>`,
			want:  `<t>Price &lt; 100</t>`,
		},
		{
			name:  "&gt; in marker decoded",
			input: `<t>{{ .Count &gt; 5 }}</t>`,
			want:  `<t>{{ .Count > 5 | xml }}</t>`,
		},
		{
			name:  "&lt; in marker decoded",
			input: `<t>{{ .Count &lt; 5 }}</t>`,
			want:  `<t>{{ .Count < 5 | xml }}</t>`,
		},

		// --- legacy markers fail with a migration hint ---

		{
			name:    "legacy tr marker",
			input:   `<t>{{tr range .Assets }}</t>`,
			wantErr: "legacy",
		},
		{
			name:    "legacy p marker",
			input:   `<t>{{p end }}</t>`,
			wantErr: "legacy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := reconstructMarkers([]byte(tt.input))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestReconstructMarkersOutputSpans(t *testing.T) {
	// These tests verify that returned marker spans index into the reconstructed
	// output bytes, not the original input. Offset correctness matters for the
	// pivot/LCA step that reads both the node tree and the marker positions from
	// the same (output) byte slice.
	tests := []struct {
		name        string
		input       string
		wantOut     string
		wantMarkers []marker
	}{
		{
			// A single-run value marker: emission is longer than the original
			// {{ }} span, so the output span is wider.
			name:    "single value marker span",
			input:   `<t>{{.X}}</t>`,
			wantOut: `<t>{{ .X | xml }}</t>`,
			wantMarkers: []marker{
				{src: span{3, 17}, text: ".X"},
			},
		},
		{
			// Second marker follows first: the output position of the second
			// must account for the expansion of the first emission.
			name:    "two single-run markers — second span shifts with first expansion",
			input:   `<t>{{.X}}</t><t>{{.Y}}</t>`,
			wantOut: `<t>{{ .X | xml }}</t><t>{{ .Y | xml }}</t>`,
			wantMarkers: []marker{
				{src: span{3, 17}, text: ".X"},
				// raw[9:16] = "</t><t>" (7 bytes) copied verbatim; output starts at 17+7=24
				{src: span{24, 38}, text: ".Y"},
			},
		},
		{
			// Cross-run marker compresses many bytes into fewer; the marker that
			// follows must shift backwards in output space accordingly.
			name:    "cross-run marker compression shifts subsequent marker",
			input:   `<t>{{range</t><t> .X}}</t><t>{{end}}</t>`,
			wantOut: `<t>{{ range .X }}</t><t>{{ end }}</t>`,
			wantMarkers: []marker{
				// original span{3,22} (19 bytes) → emission "{{ range .X }}" (14 bytes)
				{src: span{3, 17}, text: "range .X"},
				// raw[22:29] = "</t><t>" (7 bytes) → output start 17+7=24
				// emission "{{ end }}" (9 bytes) → end 24+9=33
				{src: span{24, 33}, text: "end"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, markers, err := reconstructMarkers([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, string(out))
			require.Equal(t, tt.wantMarkers, markers)

			// Cross-check: each span must bracket exactly the emitted text in out.
			for i, m := range markers {
				got := string(out[m.src.start:m.src.end])
				assert.Equal(t, buildEmission(m.text), got, "marker %d span content", i)
			}
		})
	}
}
