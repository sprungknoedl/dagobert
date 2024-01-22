package doct

import (
	"archive/zip"
	"io"
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
	for _, item := range zr.File {
		err = func() error {
			ir, err := item.Open()
			if err != nil {
				return err
			}
			defer ir.Close()

			header, err := zip.FileInfoHeader(item.FileInfo())
			if err != nil {
				return err
			}

			header.Name = item.Name
			target, err := zw.CreateHeader(header)
			if err != nil {
				return err
			}

			return fn(header, ir, target)
		}()
		if err != nil {
			return err
		}
	}

	return zw.Close()
}
