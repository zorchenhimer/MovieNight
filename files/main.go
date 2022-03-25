package files

import (
	"embed"
)

//go:embed static/*.html static/css static/img static/js
var StaticFS embed.FS
