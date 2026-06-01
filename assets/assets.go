// Package assets embeds application assets used at runtime.
package assets

import _ "embed"

// IconPNG is the full-size devtabs application icon.
//
//go:embed devtabs-icon.png
var IconPNG []byte
