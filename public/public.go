// Package public embeds the built static assets (CSS, JS, images) served by dagobert.
package public

import "embed"

//go:embed assets
var AssetsFS embed.FS
