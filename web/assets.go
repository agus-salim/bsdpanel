package webassets

import "embed"

// Assets contains all embedded web templates and static files.
//
//go:embed templates static
var Assets embed.FS
