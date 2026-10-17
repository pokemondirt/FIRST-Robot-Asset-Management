// Package webembed carries a copy of the built frontend inside the executable
// so the release is a single .exe with no dist folder to keep in sync.
//
// The copy under dist/ is produced by build-frontend.ps1 (and by the release
// workflow); the placeholder .gitkeep keeps `go build` working before the
// frontend has ever been built.
package webembed

import "embed"

// Dist holds the built UI. `all:` makes the embedding include files whose name
// starts with "_" or ".", which Vite sometimes emits.
//
//go:embed all:dist
var Dist embed.FS
