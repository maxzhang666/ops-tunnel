package main

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var frontendAssets embed.FS

func frontendFS() (fs.FS, error) {
	return fs.Sub(frontendAssets, "dist")
}
