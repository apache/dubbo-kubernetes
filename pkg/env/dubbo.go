package env

import (
	"path/filepath"
	"runtime"
)

var (
	// REPO_ROOT environment variable
	// nolint: revive, stylecheck
	REPO_ROOT Variable = "REPO_ROOT"

	DubboSrc = REPO_ROOT.ValueOrDefaultFunc(getDefaultDubboSrc)
)

var (
	_, b, _, _ = runtime.Caller(0)

	// Root folder of this project
	// This relies on the fact this file is 3 levels up from the root; if this changes, adjust the path below
	Root = filepath.Clean(filepath.Join(filepath.Dir(b), "../.."))
)

func getDefaultDubboSrc() string {
	return Root
}
