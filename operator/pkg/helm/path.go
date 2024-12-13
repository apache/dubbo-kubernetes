//go:build !windows

package helm

import (
	"path/filepath"
)

func pathJoin(elem ...string) string {
	return filepath.Join(elem...)
}
