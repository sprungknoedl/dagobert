package doct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreNormalise(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// XML character entities must NOT be replaced — raw < breaks XML parsing.
		// They are decoded later in decodeMarkerExpr on the extracted expression.
		{name: "&lt; preserved", input: `&lt;`, want: `&lt;`},
		{name: "&gt; preserved", input: `&gt;`, want: `&gt;`},
		{name: "&quot; preserved", input: `&quot;`, want: `&quot;`},

		// XML numeric character references for typographic quotes are safe to
		// replace because the resulting ASCII character is valid everywhere the
		// reference would be valid.
		{name: "&#x201C; to straight double quote", input: `&#x201C;`, want: `"`},
		{name: "&#x201D; to straight double quote", input: `&#x201D;`, want: `"`},
		{name: "&#x2018; to straight single quote", input: `&#x2018;`, want: `'`},
		{name: "&#x2019; to straight single quote", input: `&#x2019;`, want: `'`},

		// UTF-8 typographic quotes stored as raw bytes in the XML file
		{name: "UTF-8 left double quote to straight", input: "“", want: `"`},
		{name: "UTF-8 right double quote to straight", input: "”", want: `"`},
		{name: "UTF-8 left single quote to straight", input: "‘", want: `'`},
		{name: "UTF-8 right single quote to straight", input: "’", want: `'`},

		// Only typographic-quote substitutions fire; XML entities are untouched.
		{
			name:  "typographic quotes replaced, XML entities preserved",
			input: `{{ .Foo &#x201C;bar&#x201D; &lt;baz&gt; }}`,
			want:  `{{ .Foo "bar" &lt;baz&gt; }}`,
		},

		// Input without any replaceable sequences must pass through unchanged
		{name: "plain text unchanged", input: `<t>hello</t>`, want: `<t>hello</t>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preNormalise([]byte(tt.input))
			assert.Equal(t, tt.want, string(got))
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

		// Control words are structural; Go template interprets them, not the xml pipe
		{name: "range", expr: "range .Events", want: "{{ range .Events }}"},
		{name: "end", expr: "end", want: "{{ end }}"},
		{name: "if", expr: "if .Cond", want: "{{ if .Cond }}"},
		{name: "else", expr: "else", want: "{{ else }}"},
		{name: "with", expr: "with .X", want: "{{ with .X }}"},
		{name: "define", expr: `define "name"`, want: `{{ define "name" }}`},
		{name: "block", expr: `block "header" .`, want: `{{ block "header" . }}`},
		{name: "template", expr: `template "footer" .`, want: `{{ template "footer" . }}`},

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
		wantErr bool
	}{
		{
			// --- cases from the spec ---
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
			wantErr: true,
		},

		// --- additional cases ---

		{
			// Surrounding XML that contains no markers must be returned unchanged.
			name:  "no markers",
			input: `<t>plain text</t>`,
			want:  `<t>plain text</t>`,
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

		// --- regression: XML entities in document content and markers ---

		{
			// &lt; in regular document text (e.g. "Price < 100") must survive
			// intact. Before the fix, preNormalise replaced it with raw <, which
			// made the XML invalid and caused "expected element name after <".
			name:  "&lt; in document content preserved",
			input: `<t>Price &lt; 100</t>`,
			want:  `<t>Price &lt; 100</t>`,
		},
		{
			// &gt; and &lt; inside a marker must be decoded to the comparison
			// operators the template author intended.
			name:  "&gt; in marker decoded",
			input: `<t>{{ .Count &gt; 5 }}</t>`,
			want:  `<t>{{ .Count > 5 | xml }}</t>`,
		},
		{
			name:  "&lt; in marker decoded",
			input: `<t>{{ .Count &lt; 5 }}</t>`,
			want:  `<t>{{ .Count < 5 | xml }}</t>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := reconstructMarkers([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
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
