// Package assets embeds the compiled + vendored static files served at
// /public/assets/ (dagobert.css/js, third-party JS, fonts, images). See
// internal/frontend for the source inputs that build-web compiles into
// dagobert.css.
package assets

import "embed"

//go:embed *.css *.js *.woff2 *.ttf *.ico *.svg vditor-3.11.2
var FS embed.FS
