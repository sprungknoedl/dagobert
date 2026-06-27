package doct

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

type Template interface {
	Name() string
	Type() string
	Ext() string
	Render(w io.Writer, data any, funcs template.FuncMap) error
}

type Processor func(header *zip.FileHeader, r io.Reader, w io.Writer) error

func processZip(r io.ReaderAt, size int64, w io.Writer, fn Processor) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, item := range zr.File {
		err = func() error {
			ir, err := item.Open()
			if err != nil {
				return err
			}
			defer ir.Close()

			// Use a deterministic timestamp for reproducible archives
			hdr := &zip.FileHeader{
				Name:     item.Name,
				Method:   zip.Deflate,
				Modified: time.Unix(0, 0).UTC(),
			}
			target, err := zw.CreateHeader(hdr)
			if err != nil {
				return err
			}

			return fn(hdr, ir, target)
		}()
		if err != nil {
			return err
		}
	}

	return zw.Close()
}

type formatConfig struct {
	contentFile string
	mimeType    string
}

var (
	docxConfig = formatConfig{
		contentFile: "word/document.xml",
		mimeType:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}
	odtConfig = formatConfig{
		contentFile: "content.xml",
		mimeType:    "application/vnd.oasis.opendocument.text",
	}
)

type OfficeTpl struct {
	name string
	cfg  formatConfig
	src  []byte             // raw bytes of the template file, read once at load
	tpl  *template.Template // compiled once at load time
}

func xmlEscape(s string) (string, error) {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// validateXML checks that the preprocessed content is still well-formed.
// Token (unlike RawToken) verifies that start and end elements match, so
// marker reconstruction or hoisting that broke the tree fails at load time
// instead of producing corrupt documents.
func validateXML(content []byte) error {
	d := xml.NewDecoder(bytes.NewReader(content))
	for {
		if _, err := d.Token(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func load(path string, cfg formatConfig) (Template, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, err
	}

	var contentBytes []byte
	for _, item := range zr.File {
		if item.Name != cfg.contentFile {
			continue
		}
		ir, err := item.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(ir)
		ir.Close()
		if err != nil {
			return nil, err
		}

		processed, markers, err := reconstructMarkers(raw)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", path, err)
		}
		hoisted, err := hoistPivots(processed, markers)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", path, err)
		}
		if err := validateXML(hoisted); err != nil {
			return nil, fmt.Errorf("template %q: preprocessing produced invalid XML "+
				"(check for markers spanning structural boundaries): %w", path, err)
		}
		contentBytes = hoisted
	}

	funcMap := template.FuncMap{"xml": xmlEscape}
	maps.Copy(funcMap, helperFuncs)
	tpl, err := template.New(cfg.contentFile).
		Option("missingkey=error").
		Funcs(funcMap).
		Parse(string(contentBytes))
	if err != nil {
		return nil, fmt.Errorf("parsing template in %q: %w", path, err)
	}

	return OfficeTpl{
		name: filepath.Base(path),
		cfg:  cfg,
		src:  src,
		tpl:  tpl,
	}, nil
}

func LoadMsTemplate(path string) (Template, error)    { return load(path, docxConfig) }
func LoadLibreTemplate(path string) (Template, error) { return load(path, odtConfig) }

func (tpl OfficeTpl) Name() string { return tpl.name }
func (tpl OfficeTpl) Type() string { return tpl.cfg.mimeType }
func (tpl OfficeTpl) Ext() string  { return filepath.Ext(tpl.name) }

func (tpl OfficeTpl) Render(dst io.Writer, data any, funcs template.FuncMap) error {
	// The parsed template is shared/cached across renders. Per-render funcs must
	// be applied to a clone so concurrent renders don't mutate the shared one.
	render := tpl.tpl
	if funcs != nil {
		clone, err := tpl.tpl.Clone()
		if err != nil {
			return fmt.Errorf("cloning template %q: %w", tpl.name, err)
		}
		render = clone.Funcs(funcs)
	}

	return processZip(bytes.NewReader(tpl.src), int64(len(tpl.src)), dst, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == tpl.cfg.contentFile {
			if err := render.Execute(w, data); err != nil {
				return fmt.Errorf("rendering %q: %w", tpl.name, err)
			}
			return nil
		}
		_, err := io.Copy(w, r)
		return err
	})
}
