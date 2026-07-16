// Package assets embeds static vendor assets (e.g. Phosphor icons CSS) not covered by public/public.go.
package assets

import (
	_ "embed"
)

//go:embed phosphor-2.1.2.css
var PhosphorCSS string
