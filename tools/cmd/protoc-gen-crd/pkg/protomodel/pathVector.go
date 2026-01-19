package protomodel

import (
	"strconv"
)

// The SourceCodeInfo messages describes the location of elements of a parsed
// .proto currentFile by way of a "path", which is a sequence of integers that
// describe the route from a FileDescriptorProto to the relevant submessage.
// The path alternates between a field number of a repeated field, and an index
// into that repeated field. The constants below define the field numbers that
// are used.
//
// See descriptor.proto for more information about this.
const (
	// tag numbers in FileDescriptorProto
	packagePath = 2 // package
	messagePath = 4 // message_type
	enumPath    = 5 // enum_type
	servicePath = 6 // service

	// tag numbers in DescriptorProto
	messageFieldPath   = 2 // field
	messageMessagePath = 3 // nested_type
	messageEnumPath    = 4 // enum_type

	// tag numbers in EnumDescriptorProto
	enumValuePath = 2 // value

	// tag numbers in ServiceDescriptorProto
	serviceMethodPath = 2 // method
)

// A vector of comma-separated integers which identify a particular entry in a
// given's file location information
type pathVector string

func newPathVector(v int) pathVector {
	return pathVector(strconv.Itoa(v))
}

func (pv pathVector) append(v ...int) pathVector {
	result := pv
	for _, val := range v {
		result += pathVector("," + strconv.Itoa(val))
	}
	return result
}
