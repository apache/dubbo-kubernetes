package util

import "fmt"

type ErrNotInitialized struct {
	Path string
}

func NewErrNotInitialized(path string) error {
	return &ErrNotInitialized{Path: path}
}

func (e ErrNotInitialized) Error() string {
	if e.Path == "" {
		return "sdk is not initialized"
	}
	return fmt.Sprintf("'%s' does not contain an initialized sdk", e.Path)
}
