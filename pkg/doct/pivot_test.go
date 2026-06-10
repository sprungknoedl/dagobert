package doct

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyEdits(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		edits   []edit
		want    string
		wantErr bool
	}{
		{
			name:  "replace",
			raw:   "abcdef",
			edits: []edit{{src: span{2, 4}, text: "X"}},
			want:  "abXef",
		},
		{
			name:  "pure insert",
			raw:   "abcdef",
			edits: []edit{{src: span{3, 3}, text: "X"}},
			want:  "abcXdef",
		},
		{
			name:  "pure delete",
			raw:   "abcdef",
			edits: []edit{{src: span{1, 3}}},
			want:  "adef",
		},
		{
			// Inserts at the same position apply in seq (marker document)
			// order, regardless of the order they were collected in.
			name: "equal position ordered by seq",
			raw:  "ab",
			edits: []edit{
				{src: span{1, 1}, text: "2", seq: 2},
				{src: span{1, 1}, text: "1", seq: 1},
			},
			want: "a12b",
		},
		{
			// A delete and a later insert at its start position: the insert
			// (higher seq) lands before the deleted span's replacement point.
			name: "insert before delete at same position",
			raw:  "abcd",
			edits: []edit{
				{src: span{1, 1}, text: "X", seq: 0},
				{src: span{1, 3}, seq: 1},
			},
			want: "aXd",
		},
		{
			name: "overlapping edits rejected",
			raw:  "abcdef",
			edits: []edit{
				{src: span{1, 4}, text: "X", seq: 0},
				{src: span{3, 5}, text: "Y", seq: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyEdits([]byte(tt.raw), tt.edits)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestHoistPivots(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string // substring the error must contain; empty = no error
	}{
		{
			// The primary authoring pattern (old {{tr}} semantics): range and
			// end each in a DEDICATED row. The marker rows hold nothing else,
			// so they dissolve and only the data row repeats.
			name: "dedicated marker rows dissolve",
			input: `<w:tbl>` +
				`<w:tr><w:tc><w:p><w:r><w:t>Header</w:t></w:r></w:p></w:tc></w:tr>` +
				`<w:tr><w:tc><w:p><w:r><w:t>{{ range .Assets }}</w:t></w:r></w:p></w:tc></w:tr>` +
				`<w:tr><w:tc><w:p><w:r><w:t>{{ .Name }}</w:t></w:r></w:p></w:tc></w:tr>` +
				`<w:tr><w:tc><w:p><w:r><w:t>{{ end }}</w:t></w:r></w:p></w:tc></w:tr>` +
				`</w:tbl>`,
			want: `<w:tbl>` +
				`<w:tr><w:tc><w:p><w:r><w:t>Header</w:t></w:r></w:p></w:tc></w:tr>` +
				`{{ range .Assets }}` +
				`<w:tr><w:tc><w:p><w:r><w:t>{{ .Name | xml }}</w:t></w:r></w:p></w:tc></w:tr>` +
				`{{ end }}` +
				`</w:tbl>`,
		},
		{
			// Old {{p}} semantics: dedicated marker paragraphs dissolve and
			// the content paragraph between them repeats.
			name: "dedicated marker paragraphs dissolve",
			input: `<body>` +
				`<p><t>{{ range .Items }}</t></p>` +
				`<p><t>{{ .Name }}</t></p>` +
				`<p><t>{{ end }}</t></p>` +
				`</body>`,
			want: `<body>{{ range .Items }}<p><t>{{ .Name | xml }}</t></p>{{ end }}</body>`,
		},
		{
			// Markers in different cells of the SAME row: repeating the cells
			// would change the column count, so the row itself is wrapped.
			name: "same-row loop wraps the row",
			input: `<w:tbl><w:tr>` +
				`<w:tc><w:p><w:t>{{ range .Events }}</w:t></w:p></w:tc>` +
				`<w:tc><w:p><w:t>{{ end }}</w:t></w:p></w:tc>` +
				`</w:tr></w:tbl>`,
			want: `<w:tbl>{{ range .Events }}<w:tr>` +
				`<w:tc><w:p><w:t></w:t></w:p></w:tc>` +
				`<w:tc><w:p><w:t></w:t></w:p></w:tc>` +
				`</w:tr>{{ end }}</w:tbl>`,
		},
		{
			// A whole chain inside one text node is an inline loop; the text
			// around it must not repeat and nothing moves.
			name:  "inline loop in one text node untouched",
			input: `<body><p><t>before {{ range .I }}{{ .X }}, {{ end }} after</t></p></body>`,
			want:  `<body><p><t>before {{ range .I }}{{ .X | xml }}, {{ end }} after</t></p></body>`,
		},
		{
			// Markers already sitting between elements are at the right level.
			name:  "already correct placement untouched",
			input: `<body>{{ range .Items }}<item>data</item>{{ end }}</body>`,
			want:  `<body>{{ range .Items }}<item>data</item>{{ end }}</body>`,
		},
		{
			// Marker-only runs inside one paragraph dissolve; the content run
			// repeats inline within the paragraph.
			name: "marker-only runs dissolve to inline loop",
			input: `<p>` +
				`<r><t>{{ range .X }}</t></r>` +
				`<r><t>item</t></r>` +
				`<r><t>{{ end }}</t></r>` +
				`</p>`,
			want: `<p>{{ range .X }}<r><t>item</t></r>{{ end }}</p>`,
		},
		{
			// Marker-only spans of an ODT paragraph dissolve the same way.
			name: "marker-only spans dissolve",
			input: `<office:body>` +
				`<text:p><text:span>{{ range .Items }}</text:span><text:span>{{ end }}</text:span></text:p>` +
				`</office:body>`,
			want: `<office:body><text:p>{{ range .Items }}{{ end }}</text:p></office:body>`,
		},
		{
			// Full if/else/end chain across dedicated paragraphs: each branch
			// wraps complete sibling elements, so both branches stay balanced.
			name: "if else end in dedicated paragraphs",
			input: `<body>` +
				`<p><t>{{ if .X }}</t></p>` +
				`<p><t>then</t></p>` +
				`<p><t>{{ else }}</t></p>` +
				`<p><t>else</t></p>` +
				`<p><t>{{ end }}</t></p>` +
				`</body>`,
			want: `<body>{{ if .X }}<p><t>then</t></p>{{ else }}<p><t>else</t></p>{{ end }}</body>`,
		},
		{
			// else-if chains hoist every branch boundary.
			name: "else if chain",
			input: `<body>` +
				`<p><t>{{ if .A }}</t></p>` +
				`<p><t>a</t></p>` +
				`<p><t>{{ else if .B }}</t></p>` +
				`<p><t>b</t></p>` +
				`<p><t>{{ end }}</t></p>` +
				`</body>`,
			want: `<body>{{ if .A }}<p><t>a</t></p>{{ else if .B }}<p><t>b</t></p>{{ end }}</body>`,
		},
		{
			// A single row cannot be split between two branches.
			name: "else inside row guard is an error",
			input: `<w:tbl><w:tr>` +
				`<w:tc><w:t>{{ if .X }}</w:t></w:tc>` +
				`<w:tc><w:t>{{ else }}</w:t></w:tc>` +
				`<w:tc><w:t>{{ end }}</w:t></w:tc>` +
				`</w:tr></w:tbl>`,
			wantErr: "cannot be hoisted inside a single table row",
		},
		{
			// Adjacent loops: loop 1's {{ end }} and loop 2's {{ range }} are
			// inserted at the same byte position and must come out in marker
			// document order, not swapped.
			name: "adjacent sibling loops keep order",
			input: `<w:tbl>` +
				`<w:tr><w:tc><w:t>{{ range .A }}</w:t></w:tc><w:tc><w:t>x{{ end }}</w:t></w:tc></w:tr>` +
				`<w:tr><w:tc><w:t>{{ range .B }}</w:t></w:tc><w:tc><w:t>y{{ end }}</w:t></w:tc></w:tr>` +
				`</w:tbl>`,
			want: `<w:tbl>` +
				`{{ range .A }}<w:tr><w:tc><w:t></w:t></w:tc><w:tc><w:t>x</w:t></w:tc></w:tr>{{ end }}` +
				`{{ range .B }}<w:tr><w:tc><w:t></w:t></w:tc><w:tc><w:t>y</w:t></w:tc></w:tr>{{ end }}` +
				`</w:tbl>`,
		},
		{
			// Two chains wrapping the same row: openers nest in document order
			// and the inner end closes before the outer end.
			name: "nested chains on the same row",
			input: `<w:tbl><w:tr>` +
				`<w:tc><w:t>{{ range .A }}{{ if .B }}</w:t></w:tc>` +
				`<w:tc><w:t>{{ end }}{{ end }}</w:t></w:tc>` +
				`</w:tr></w:tbl>`,
			want: `<w:tbl>{{ range .A }}{{ if .B }}<w:tr>` +
				`<w:tc><w:t></w:t></w:tc>` +
				`<w:tc><w:t></w:t></w:tc>` +
				`</w:tr>{{ end }}{{ end }}</w:tbl>`,
		},
		{
			// A dedicated paragraph shared by two chains ({{ end }} of the
			// first loop and {{ range }} of the second) dissolves once, with
			// both markers emitted in document order.
			name: "shared dissolved anchor",
			input: `<body>` +
				`<p><t>{{ range .A }}</t></p>` +
				`<p><t>a</t></p>` +
				`<p><t>{{ end }}{{ range .B }}</t></p>` +
				`<p><t>b</t></p>` +
				`<p><t>{{ end }}</t></p>` +
				`</body>`,
			want: `<body>` +
				`{{ range .A }}<p><t>a</t></p>{{ end }}` +
				`{{ range .B }}<p><t>b</t></p>{{ end }}` +
				`</body>`,
		},
		{
			// An OOXML cell must keep at least one paragraph: the only block
			// child of a cell is never dissolved, the marker moves beside it.
			name: "only paragraph in cell is kept",
			input: `<body><w:tr><w:tc>` +
				`<w:p><w:t>{{ range .X }}</w:t></w:p>{{ end }}` +
				`</w:tc></w:tr></body>`,
			want: `<body><w:tr><w:tc>` +
				`{{ range .X }}<w:p><w:t></w:t></w:p>{{ end }}` +
				`</w:tc></w:tr></body>`,
		},
		{
			// ODF row guard: table:table-row covers Writer tables and Calc sheets.
			name: "ODF same-row loop wraps the row",
			input: `<table:table><table:table-row>` +
				`<table:table-cell><text:p>{{ range .R }}</text:p></table:table-cell>` +
				`<table:table-cell><text:p>x{{ end }}</text:p></table:table-cell>` +
				`</table:table-row></table:table>`,
			want: `<table:table>{{ range .R }}<table:table-row>` +
				`<table:table-cell><text:p></text:p></table:table-cell>` +
				`<table:table-cell><text:p>x</text:p></table:table-cell>` +
				`</table:table-row>{{ end }}</table:table>`,
		},
		{
			// ODF dedicated marker rows dissolve like their OOXML counterpart.
			name: "ODF dedicated marker rows dissolve",
			input: `<table:table>` +
				`<table:table-row><table:table-cell><text:p>{{ range .R }}</text:p></table:table-cell></table:table-row>` +
				`<table:table-row><table:table-cell><text:p>{{ .V }}</text:p></table:table-cell></table:table-row>` +
				`<table:table-row><table:table-cell><text:p>{{ end }}</text:p></table:table-cell></table:table-row>` +
				`</table:table>`,
			want: `<table:table>` +
				`{{ range .R }}` +
				`<table:table-row><table:table-cell><text:p>{{ .V | xml }}</text:p></table:table-cell></table:table-row>` +
				`{{ end }}` +
				`</table:table>`,
		},
		{
			// Nested {{ if }} inside {{ range }}: the outer chain hoists to
			// row boundaries (rows hold data, so they are kept, not dissolved)
			// while the inner chain wraps its own row. Depth tracking ensures
			// the middle {{ end }} closes the if, not the range.
			name: "nested if inside range",
			input: `<body>` +
				`<w:tr><cell><t>{{ range .Events }}</t></cell><cell><t>data</t></cell></w:tr>` +
				`<w:tr><cell><t>{{ if .Flag }}</t></cell><cell><t>{{ end }}</t></cell></w:tr>` +
				`<w:tr><cell><t>{{ end }}</t></cell><cell><t>data</t></cell></w:tr>` +
				`</body>`,
			want: `<body>` +
				`{{ range .Events }}<w:tr><cell><t></t></cell><cell><t>data</t></cell></w:tr>` +
				`{{ if .Flag }}<w:tr><cell><t></t></cell><cell><t></t></cell></w:tr>{{ end }}` +
				`<w:tr><cell><t></t></cell><cell><t>data</t></cell></w:tr>{{ end }}` +
				`</body>`,
		},
		{
			// Markers outside the document element have no common ancestor.
			name:    "mismatched markers root LCA",
			input:   `{{ range .X }}<doc><p>content</p></doc>{{ end }}`,
			wantErr: "no common ancestor",
		},
		{
			name:    "unclosed opener",
			input:   `<body><p><t>{{ range .X }}</t></p></body>`,
			wantErr: "unclosed {{ range }}",
		},
		{
			name:  "no markers",
			input: `<body><p>plain text</p></body>`,
			want:  `<body><p>plain text</p></body>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build markers via reconstructMarkers so spans match the raw bytes.
			raw, markers, err := reconstructMarkers([]byte(tt.input))
			require.NoError(t, err)

			got, err := hoistPivots(raw, markers)
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
