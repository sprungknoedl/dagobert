package doct

import (
	"archive/zip"
	"io"
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
