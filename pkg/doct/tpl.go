package doct

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

type Template interface {
	Name() string
	Type() string
	Ext() string
	Render(w io.Writer, data interface{}) error
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
	src  io.ReaderAt
	len  int64
	tpl  *template.Template
}

func xmlEscape(s string) (string, error) {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func load(path string, cfg formatConfig) (Template, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var contentBytes []byte
	buf := new(bytes.Buffer)
	err = processZip(fh, stat.Size(), buf, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == cfg.contentFile {
			raw, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			processed, _, err := reconstructMarkers(raw)
			if err != nil {
				return err
			}
			contentBytes = processed
			_, err = w.Write(processed)
			return err
		}
		_, err := io.Copy(w, r)
		return err
	})
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{"xml": xmlEscape}
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
		src:  bytes.NewReader(buf.Bytes()),
		len:  int64(buf.Len()),
		tpl:  tpl,
	}, nil
}

func LoadMsTemplate(path string) (Template, error)    { return load(path, docxConfig) }
func LoadLibreTemplate(path string) (Template, error) { return load(path, odtConfig) }

func (tpl OfficeTpl) Name() string { return tpl.name }
func (tpl OfficeTpl) Type() string { return tpl.cfg.mimeType }
func (tpl OfficeTpl) Ext() string  { return filepath.Ext(tpl.name) }

func (tpl OfficeTpl) Render(dst io.Writer, data interface{}) error {
	return processZip(tpl.src, tpl.len, dst, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == tpl.cfg.contentFile {
			if err := tpl.tpl.Execute(w, data); err != nil {
				return fmt.Errorf("rendering %q: %w", tpl.name, err)
			}
			return nil
		}
		_, err := io.Copy(w, r)
		return err
	})
}
