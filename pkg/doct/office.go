package doct

import (
	"archive/zip"
	"bytes"
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

	buf := new(bytes.Buffer)
	err = processZip(fh, stat.Size(), buf, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == cfg.contentFile {
			return cfg.preprocess(w, r)
		}
		_, err := io.Copy(w, r)
		return err
	})
	if err != nil {
		return nil, err
	}

	return OfficeTpl{
		name: filepath.Base(path),
		cfg:  cfg,
		src:  bytes.NewReader(buf.Bytes()),
		len:  int64(buf.Len()),
	}, nil
}

func (tpl OfficeTpl) Name() string { return tpl.name }
func (tpl OfficeTpl) Type() string { return tpl.cfg.mimeType }
func (tpl OfficeTpl) Ext() string  { return filepath.Ext(tpl.name) }

func (tpl OfficeTpl) Render(dst io.Writer, data interface{}) error {
	return processZip(tpl.src, tpl.len, dst, func(header *zip.FileHeader, r io.Reader, w io.Writer) error {
		if header.Name == tpl.cfg.contentFile {
			b, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			t, err := template.New(header.Name).Parse(string(b))
			if err != nil {
				return err
			}

			return t.Execute(w, data)
		}
		_, err := io.Copy(w, r)
		return err
	})
}
