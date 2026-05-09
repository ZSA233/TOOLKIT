//go:build mtu_tuner_embed_frontend

package main

import (
	"embed"
	"io/fs"
)

//go:embed frontend/dist
var embeddedFrontendDist embed.FS

func frontendAssetFS() (fs.FS, error) {
	return fs.Sub(embeddedFrontendDist, "frontend/dist")
}
