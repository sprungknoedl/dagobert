package doct

import (
	"bytes"
	"io"

	"go.arsenm.dev/pcre"
)

var (
	libre_pRegexp   = pcre.MustCompile(`<text:p[^>]*?>(?:(?!<text:p[ >]).)*{{p (.+?)}}.*?<\/text:p>`)
	libre_trRegexp  = pcre.MustCompile(`<table:table-row[^>]*>(?:(?!<table:table-row[ >]).)*{{tr (.+?)}}.*?<\/table:table-row>`)
	libre_expRegexp = pcre.MustCompile(`{{([^}]+)}}`)

	// replace {<something>{ by {{   ( works with {{ }} {% and %} {# and #})
	libre_clean1Regexp = pcre.MustCompile(`(?<={)(<[^>]*>)+(?=[\{%\#])|(?<=[%\}\#])(<[^>]*>)+(?=\})`)

	// replace {{<some tags>go stuff<some other tags>}} by {{go stuff}}
	libre_clean2Regexp    = pcre.MustCompile(`{%(?:(?!%}).)*|{#(?:(?!#}).)*|{{(?:(?!}}).)*`)
	libre_clean2SubRegexp = pcre.MustCompile(`<\/?text:span[^>]*>`)
)

func LoadLibreTemplate(path string) (Template, error) { return load(path, odtConfig) }

func preprocessLibreContent(w io.Writer, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// replace {<something>{ by {{ ( works with {{ }} {% and %} {# and #})
	b = libre_clean1Regexp.ReplaceAll(b, nil)

	// replace {{<some tags>go stuff<some other tags>}} by {{go stuff}}
	b = libre_clean2Regexp.ReplaceAllFunc(b, func(x []byte) []byte {
		return libre_clean2SubRegexp.ReplaceAll(x, nil)
	})

	// replace into xml code the paragraph containing
	// {{p xxx }} template tag by {{ xxx }} without any surrounding
	// <text:p> tags
	b = libre_pRegexp.ReplaceAll(b, []byte("{{ $1 }}"))

	// replace into xml code the table row containing
	// {{tr xxx }} template tag by {{ xxx }} without any surrounding
	// <table:table-row> tags
	b = libre_trRegexp.ReplaceAll(b, []byte("{{ $1 }}"))

	// clean tags
	b = libre_expRegexp.ReplaceAllFunc(b, func(x []byte) []byte {
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
