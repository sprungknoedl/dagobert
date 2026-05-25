package doct

import (
	"bytes"
	"io"

	"go.arsenm.dev/pcre"
)

var (
	ms_pRegexp   = pcre.MustCompile(`<w:p[^>]*?>(?:(?!<w:p[ >]).)*{{p (.+?)}}.*?<\/w:p>`)
	ms_trRegexp  = pcre.MustCompile(`<w:tr[^>]*>(?:(?!<w:tr[ >]).)*{{tr (.+?)}}.*?<\/w:tr>`)
	ms_expRegexp = pcre.MustCompile(`{{([^}]+)}}`)

	// replace {<something>{ by {{   ( works with {{ }} {% and %} {# and #})
	ms_clean1Regexp = pcre.MustCompile(`(?<={)(<[^>]*>)+(?=[\{%\#])|(?<=[%\}\#])(<[^>]*>)+(?=\})`)

	// replace {{<some tags>go stuff<some other tags>}} by {{go stuff}}
	ms_clean2Regexp    = pcre.MustCompile(`{%(?:(?!%}).)*|{#(?:(?!#}).)*|{{(?:(?!}}).)*`)
	ms_clean2SubRegexp = pcre.MustCompile(`<\/?w:t[^>]*>`)
)

func LoadMsTemplate(path string) (Template, error) { return load(path, docxConfig) }

func preprocessMsContent(w io.Writer, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// replace {<something>{ by {{ ( works with {{ }} {% and %} {# and #})
	b = ms_clean1Regexp.ReplaceAll(b, nil)

	// replace {{<some tags>go stuff<some other tags>}} by {{go stuff}}
	b = ms_clean2Regexp.ReplaceAllFunc(b, func(x []byte) []byte {
		return ms_clean2SubRegexp.ReplaceAll(x, nil)
	})

	// replace into xml code the paragraph containing
	// {{p xxx }} template tag by {{ xxx }} without any surrounding
	// <text:p> tags
	b = ms_pRegexp.ReplaceAll(b, []byte("{{ $1 }}"))

	// replace into xml code the table row containing
	// {{tr xxx }} template tag by {{ xxx }} without any surrounding
	// <table:table-row> tags
	b = ms_trRegexp.ReplaceAll(b, []byte("{{ $1 }}"))

	// clean tags
	b = ms_expRegexp.ReplaceAllFunc(b, func(x []byte) []byte {
		x = bytes.ReplaceAll(x, []byte("&quot;"), []byte("\""))
		x = bytes.ReplaceAll(x, []byte("&lt;"), []byte("<"))
		x = bytes.ReplaceAll(x, []byte("&gt;"), []byte(">"))
		x = bytes.ReplaceAll(x, []byte("“"), []byte("\""))
		x = bytes.ReplaceAll(x, []byte("”"), []byte("\""))
		x = bytes.ReplaceAll(x, []byte("‘"), []byte("'"))
		x = bytes.ReplaceAll(x, []byte("’"), []byte("'"))
		return x
	})

	_, err = w.Write(b)
	return err
}
