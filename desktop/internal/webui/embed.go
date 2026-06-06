// Package webui embeds the compiled React remote into the binary, so the desktop app ships as a
// single self-contained executable with no external web assets to deploy.
//
// The `dist` directory is populated by `scripts/build-web.sh` (which runs `vite build` and copies
// the output here). A placeholder index.html is committed so the package always compiles even
// before the web bundle has been built.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// Assets returns the SPA filesystem rooted at the bundle's top level.
func Assets() (fs.FS, error) {
	return fs.Sub(files, "dist")
}
