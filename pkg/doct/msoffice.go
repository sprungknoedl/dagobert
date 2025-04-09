package doct

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"go.arsenm.dev/pcre"
)

const oxmlMainFile = "word/document.xml"

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

type MsTemplate struct {
	name string
	src  io.ReaderAt
	len  int64
}

func LoadMsTemplate(path string) (Template, error) {
	buf := new(bytes.Buffer)
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	err = processZip(fh, stat.Size(), buf, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == oxmlMainFile {
			// preprocess xml to transform it into a valid text/template AND docx document
			err = preprocessMsContent(w, r)
			return err
		} else {
			// just copy all other files
			_, err = io.Copy(w, r)
			return err
		}
	})
	if err != nil {
		return nil, err
	}

	return MsTemplate{
		name: filepath.Base(path),
		src:  bytes.NewReader(buf.Bytes()),
		len:  int64(buf.Len()),
	}, nil
}

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

func (tpl MsTemplate) Name() string {
	return tpl.name
}

func (tpl MsTemplate) Type() string {
	return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

func (tpl MsTemplate) Ext() string {
	return filepath.Ext(tpl.name)
}

func (tpl MsTemplate) Render(dst io.Writer, data interface{}) error {
	err := processZip(tpl.src, tpl.len, dst, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == oxmlMainFile {
			b, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			// process xml with text/template
			tpl, err := template.New(header.Name).Parse(string(b))
			if err != nil {
				return err
			}

			err = tpl.Execute(w, data)
			return err
		} else {
			// just copy all other files
			_, err := io.Copy(w, r)
			return err
		}
	})
	return err
}
