package protomodel

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type EnumDescriptor struct {
	baseDesc
	*descriptor.EnumDescriptorProto
	Values []*EnumValueDescriptor // The values of this enum
}

type EnumValueDescriptor struct {
	baseDesc
	*descriptor.EnumValueDescriptorProto
}

func newEnumDescriptor(desc *descriptor.EnumDescriptorProto, parent *MessageDescriptor, file *FileDescriptor, path pathVector) *EnumDescriptor {
	var qualifiedName []string
	if parent == nil {
		qualifiedName = []string{desc.GetName()}
	} else {
		qualifiedName = make([]string, len(parent.QualifiedName()), len(parent.QualifiedName())+1)
		copy(qualifiedName, parent.QualifiedName())
		qualifiedName = append(qualifiedName, desc.GetName())
	}

	e := &EnumDescriptor{
		EnumDescriptorProto: desc,
		baseDesc:            newBaseDesc(file, path, qualifiedName),
	}

	e.Values = make([]*EnumValueDescriptor, 0, len(desc.Value))
	for i, ev := range desc.Value {
		nameCopy := make([]string, len(qualifiedName), len(qualifiedName)+1)
		copy(nameCopy, qualifiedName)
		nameCopy = append(nameCopy, ev.GetName())

		evd := &EnumValueDescriptor{
			EnumValueDescriptorProto: ev,
			baseDesc:                 newBaseDesc(file, path.append(enumValuePath, i), nameCopy),
		}
		e.Values = append(e.Values, evd)
	}

	return e
}
