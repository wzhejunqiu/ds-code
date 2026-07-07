// Package assets embeds the built desktop frontend for Wails production builds.
package assets

import "embed"

// Dist contains Vite production output (desktop/frontend → ../assets/dist).
//
//go:embed all:dist
var Dist embed.FS
