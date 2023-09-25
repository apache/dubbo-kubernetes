package deploy

import "embed"

//go:embed all:*
var EmbedRootFS embed.FS
