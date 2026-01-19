package protomodel

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type MessageDescriptor struct {
	baseDesc
	*descriptor.DescriptorProto
	Parent   *MessageDescriptor   // The containing message, if any
	Messages []*MessageDescriptor // Inner messages, if any
	Enums    []*EnumDescriptor    // Inner enums, if any
	Fields   []*FieldDescriptor   // Fields, if any
}

type FieldDescriptor struct {
	baseDesc
	*descriptor.FieldDescriptorProto
	FieldType CoreDesc // Type of data held by this field
}

func newMessageDescriptor(desc *descriptor.DescriptorProto, parent *MessageDescriptor, file *FileDescriptor, path pathVector) *MessageDescriptor {
	var qualifiedName []string
	if parent == nil {
		qualifiedName = []string{desc.GetName()}
	} else {
		qualifiedName = make([]string, len(parent.QualifiedName()), len(parent.QualifiedName())+1)
		copy(qualifiedName, parent.QualifiedName())
		qualifiedName = append(qualifiedName, desc.GetName())
	}

	m := &MessageDescriptor{
		DescriptorProto: desc,
		Parent:          parent,
		baseDesc:        newBaseDesc(file, path, qualifiedName),
	}

	for i, f := range desc.Field {
		nameCopy := make([]string, len(qualifiedName), len(qualifiedName)+1)
		copy(nameCopy, qualifiedName)
		nameCopy = append(nameCopy, f.GetName())

		fd := &FieldDescriptor{
			FieldDescriptorProto: f,
			baseDesc:             newBaseDesc(file, path.append(messageFieldPath, i), nameCopy),
		}

		m.Fields = append(m.Fields, fd)
	}

	for i, msg := range desc.NestedType {
		m.Messages = append(m.Messages, newMessageDescriptor(msg, m, file, path.append(messageMessagePath, i)))
	}

	for i, e := range desc.EnumType {
		m.Enums = append(m.Enums, newEnumDescriptor(e, m, file, path.append(messageEnumPath, i)))
	}

	return m
}

func (f *FieldDescriptor) IsRepeated() bool {
	return f.Label != nil && *f.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED
}
