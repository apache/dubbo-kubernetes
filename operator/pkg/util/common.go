package util

import "strings"

func IsFilePath(path string) bool {
	return strings.Contains(path, "/") || strings.Contains(path, ".")
}
