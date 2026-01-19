package protomodel

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type LocationDescriptor struct {
	*descriptor.SourceCodeInfo_Location
	File *FileDescriptor
}

func newLocationDescriptor(desc *descriptor.SourceCodeInfo_Location, file *FileDescriptor) LocationDescriptor {
	return LocationDescriptor{
		SourceCodeInfo_Location: desc,
		File:                    file,
	}
}
