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

type OxmlTemplate struct {
	name string
	src  io.ReaderAt
	len  int64
}

func LoadOxmlTemplate(path string) (Template, error) {
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
			err = preprocessOxmlContent(w, r)
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

	return OxmlTemplate{
		name: filepath.Base(path),
		src:  bytes.NewReader(buf.Bytes()),
		len:  int64(buf.Len()),
	}, nil
}

func preprocessOxmlContent(w io.Writer, r io.Reader) error {
	var pRegexp = pcre.MustCompile(`<w:p[^>]*?>(?:(?!<w:p[ >]).)*{{p (.+?)}}.*?<\/w:p>`)
	var trRegexp = pcre.MustCompile(`<w:tr[^>]*>(?:(?!<w:tr[ >]).)*{{tr (.+?)}}.*?<\/w:tr>`)
	var expRegexp = pcre.MustCompile(`{{([^}]+)}}`)

	// replace {<something>{ by {{   ( works with {{ }} {% and %} {# and #})
	var clean1Regexp = pcre.MustCompile(`(?<={)(<[^>]*>)+(?=[\{%\#])|(?<=[%\}\#])(<[^>]*>)+(?=\})`)

	// replace {{<some tags>go stuff<some other tags>}} by {{go stuff}}
	var clean2Regexp = pcre.MustCompile(`{%(?:(?!%}).)*|{#(?:(?!#}).)*|{{(?:(?!}}).)*`)
	var clean2SubRegexp = pcre.MustCompile(`<\/?w:t[^>]*>`)

	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// replace {<something>{ by {{ ( works with {{ }} {% and %} {# and #})
	b = clean1Regexp.ReplaceAll(b, nil)

	// replace {{<some tags>go stuff<some other tags>}} by {{go stuff}}
	b = clean2Regexp.ReplaceAllFunc(b, func(x []byte) []byte {
		return clean2SubRegexp.ReplaceAll(x, nil)
	})

	// replace into xml code the paragraph containing
	// {{p xxx }} template tag by {{ xxx }} without any surrounding
	// <text:p> tags
	b = pRegexp.ReplaceAll(b, []byte("{{ $1 }}"))

	// replace into xml code the table row containing
	// {{tr xxx }} template tag by {{ xxx }} without any surrounding
	// <table:table-row> tags
	b = trRegexp.ReplaceAll(b, []byte("{{ $1 }}"))

	// clean tags
	b = expRegexp.ReplaceAllFunc(b, func(x []byte) []byte {
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

func (tpl OxmlTemplate) Name() string {
	return tpl.name
}

func (tpl OxmlTemplate) Type() string {
	return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

func (tpl OxmlTemplate) Ext() string {
	return filepath.Ext(tpl.name)
}

func (tpl OxmlTemplate) Render(dst io.Writer, data interface{}) error {
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
