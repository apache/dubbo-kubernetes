//go:build !linux
// +build !linux

package file

import (
	"os"
)

func markNotNeeded(in *os.File) error {
	return nil
}
