package protomodel

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type ServiceDescriptor struct {
	baseDesc
	*descriptor.ServiceDescriptorProto
	Methods []*MethodDescriptor // Methods, if any
}

type MethodDescriptor struct {
	baseDesc
	*descriptor.MethodDescriptorProto
	Input  *MessageDescriptor
	Output *MessageDescriptor
}

func newServiceDescriptor(desc *descriptor.ServiceDescriptorProto, file *FileDescriptor, path pathVector) *ServiceDescriptor {
	qualifiedName := []string{desc.GetName()}

	s := &ServiceDescriptor{
		ServiceDescriptorProto: desc,
		baseDesc:               newBaseDesc(file, path, qualifiedName),
	}

	for i, m := range desc.Method {
		nameCopy := make([]string, len(qualifiedName), len(qualifiedName)+1)
		copy(nameCopy, qualifiedName)
		nameCopy = append(nameCopy, m.GetName())

		md := &MethodDescriptor{
			MethodDescriptorProto: m,
			baseDesc:              newBaseDesc(file, path.append(serviceMethodPath, i), nameCopy),
		}
		s.Methods = append(s.Methods, md)
	}

	return s
}
