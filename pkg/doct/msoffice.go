package doct

import (
	"archive/zip"
	"bytes"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"go.arsenm.dev/pcre"
)

const docxMainFile = "document.xml"

type DocxTemplate struct {
	name string
	src  io.ReaderAt
	len  int64
}

type PpptTemplate struct{}

type XlsxTemplate struct{}

func LoadDocxTemplate(path string) (Template, error) {
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
		if header.Name == docxMainFile {
			// preprocess xml to transform it into a valid text/template AND docx document
			err = preprocessDocxContent(w, r)
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

	return DocxTemplate{
		name: filepath.Base(path),
		src:  bytes.NewReader(buf.Bytes()),
		len:  int64(buf.Len()),
	}, nil
}

func preprocessDocxContent(w io.Writer, r io.Reader) error {
	var pRegexp = pcre.MustCompile(`<w:p[^>]*?>{{p (.+?)}}<\/w:p>`)
	var trRegexp = pcre.MustCompile(`<w:tr[^>]*>(?:(?!<table:table-row).)*{{tr (.+?)}}.*?<\/w:tr>`)
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

func (tpl DocxTemplate) Name() string {
	return tpl.name
}

func (tpl DocxTemplate) Type() string {
	return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

func (tpl DocxTemplate) Ext() string {
	return "docx"
}

func (tpl DocxTemplate) Render(w io.Writer, data interface{}) error {
	err := processZip(tpl.src, tpl.len, w, func(header *zip.FileHeader, r2 io.Reader, w2 io.Writer) error {
		if header.Name == "document.xml" {
			b, err := io.ReadAll(r2)
			if err != nil {
				return err
			}

			// process xml with text/template
			tpl, err := template.New(header.Name).Parse(string(b))
			if err != nil {
				return err
			}

			err = tpl.Execute(w2, data)
			return err
		} else {
			// just copy all other files
			_, err := io.Copy(w2, r2)
			return err
		}
	})
	return err
}