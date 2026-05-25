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
)

type formatConfig struct {
	contentFile string
	mimeType    string
	preprocess  func(w io.Writer, r io.Reader) error
}

var (
	docxConfig = formatConfig{
		contentFile: "word/document.xml",
		mimeType:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		preprocess:  preprocessMsContent,
	}
	odtConfig = formatConfig{
		contentFile: "content.xml",
		mimeType:    "application/vnd.oasis.opendocument.text",
		preprocess:  preprocessLibreContent,
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
			var content bytes.Buffer
			err := cfg.preprocess(io.MultiWriter(w, &content), r)
			contentBytes = content.Bytes()
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
