package resources

import (
	"embed"
	"io/fs"
)

//go:embed data/*
var guiData embed.FS

func FS() fs.FS {
	fsys, err := fs.Sub(guiData, "data")
	if err != nil {
		panic(err)
	}
	return fsys
}
