package main

import "embed"

// frontendDist embeds the built frontend assets (HTML, JS, CSS).
// The dist/ directory is populated by the build process before go build:
//
//	npm run build (in frontend/)
//	cp -r frontend/dist backend/cmd/songmartyn/dist
//
// In dev mode (-dev), files are served from disk instead for hot-reload.
//
//go:embed all:dist
var frontendDist embed.FS
