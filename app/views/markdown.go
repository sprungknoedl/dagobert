package views

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark"
)

func markdown(source string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		if err := goldmark.Convert([]byte(source), w); err != nil {
			return err
		}

		return nil
	})
}
