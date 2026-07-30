// Package frontend holds the source inputs consumed by `make build-web`
// (Tailwind CSS entry point, daisyUI plugin files, vendor CSS partials) that
// compile into internal/assets/dagobert.css. Phosphor's icon-font CSS
// is one of those inputs but is additionally embedded here and parsed at
// runtime to build the icon name -> glyph map used by the network
// visualization.
package frontend

import _ "embed"

//go:embed phosphor-2.1.2.css
var PhosphorCSS string
