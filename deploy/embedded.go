package deploy

import "embed"

//go:embed all:addons all:charts all:profiles
var EmbedRootFS embed.FS
