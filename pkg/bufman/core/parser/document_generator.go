// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"strings"
)

import (
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/bufbuild/protocompile/walk"

	"google.golang.org/protobuf/reflect/protoreflect"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/bufpkg/bufmodule/bufmoduleprotocompile"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

type DocumentGenerator interface {
	GetModulePackages() []string
	GenerateDocument(packageName string) *registryv1alpha1.PackageDocumentation
}

type documentGeneratorImpl struct {
	modulePackageSet      map[string]struct{}
	commitName            string // 当前的commitName，用于判断是否是外部依赖
	packageLinkerMap      map[string]linker.Files
	linkers               linker.Files
	packageLinkers        linker.Files
	parserAccessorHandler bufmoduleprotocompile.ParserAccessorHandler
	messageSet            map[string]*registryv1alpha1.Message
}

func NewDocumentGenerator(commitName string, linkers linker.Files, parserAccessorHandler bufmoduleprotocompile.ParserAccessorHandler) DocumentGenerator {
	g := &documentGeneratorImpl{
		commitName:            commitName,
		linkers:               linkers,
		parserAccessorHandler: parserAccessorHandler,
		packageLinkerMap:      map[string]linker.Files{},
		messageSet:            map[string]*registryv1alpha1.Message{},
	}

	// 生成module packages Set
	packagesSet := map[string]struct{}{}
	for _, link := range linkers {
		packagesSet[string(link.Package())] = struct{}{}
	}
	g.modulePackageSet = packagesSet

	return g
}

// isDependency 判断是否是外部依赖
func (g *documentGeneratorImpl) isDependency(fileDescriptor protoreflect.FileDescriptor) bool {
	return g.parserAccessorHandler.Commit(fileDescriptor.Path()) != g.commitName
}

func (g *documentGeneratorImpl) toProtoLocation(loc protoreflect.SourceLocation) *registryv1alpha1.Location {
	return &registryv1alpha1.Location{
		StartLine:   int32(loc.StartLine),
		StartColumn: int32(loc.StartColumn),
		EndLine:     int32(loc.EndLine),
		EndColumn:   int32(loc.EndColumn),
	}
}

func (g *documentGeneratorImpl) getPackageLinkers(packageName string) linker.Files {
	if packageLinkers, ok := g.packageLinkerMap[packageName]; ok {
		return packageLinkers
	}

	packageLinkers := make(linker.Files, 0, len(g.linkers))
	for i := 0; i < len(g.linkers); i++ {
		if string(g.linkers[i].Package()) == packageName {
			packageLinkers = append(packageLinkers, g.linkers[i])
		}
	}
	g.packageLinkerMap[packageName] = packageLinkers
	return packageLinkers
}

func (g *documentGeneratorImpl) GetModulePackages() []string {
	modulePackages := make([]string, 0, len(g.modulePackageSet))
	for packageName := range g.modulePackageSet {
		modulePackages = append(modulePackages, packageName)
	}

	return modulePackages
}

func (g *documentGeneratorImpl) GenerateDocument(packageName string) *registryv1alpha1.PackageDocumentation {
	g.messageSet = map[string]*registryv1alpha1.Message{}

	g.packageLinkers = g.getPackageLinkers(packageName)

	return g.doGenerateDocument(packageName)
}

func (g *documentGeneratorImpl) doGenerateDocument(packageName string) *registryv1alpha1.PackageDocumentation {
	var messages []*registryv1alpha1.Message
	var enums []*registryv1alpha1.Enum
	var services []*registryv1alpha1.Service

	for i := 0; i < len(g.packageLinkers); i++ {
		packageLink := g.packageLinkers[i]

		_ = walk.Descriptors(packageLink, func(descriptor protoreflect.Descriptor) error {
			switch descriptor.(type) {
			case protoreflect.MessageDescriptor:
				messages = append(messages, g.GetMessage(descriptor.(protoreflect.MessageDescriptor)))
			case protoreflect.EnumDescriptor:
				enums = append(enums, g.GetEnum(descriptor.(protoreflect.EnumDescriptor)))
			case protoreflect.ServiceDescriptor:
				services = append(services, g.GetService(descriptor.(protoreflect.ServiceDescriptor)))
			}

			return nil
		})
	}

	return &registryv1alpha1.PackageDocumentation{
		Name: packageName,
		// TODO Description:    "",
		Services: services,
		Enums:    enums,
		Messages: messages,
		// TODO FileExtensions: nil,
	}
}

func (g *documentGeneratorImpl) getNestedName(fullName, packageName string) string {
	return strings.Replace(fullName, packageName+".", "", 1)
}

func (g *documentGeneratorImpl) GetPackageMessages() []*registryv1alpha1.Message {
	var messages []*registryv1alpha1.Message

	for i := 0; i < len(g.packageLinkers); i++ {
		packageLink := g.packageLinkers[i]
		messages = append(messages, g.GetMessages(packageLink.Messages())...)
	}

	return messages
}

func (g *documentGeneratorImpl) GetMessages(messageDescriptors protoreflect.MessageDescriptors) []*registryv1alpha1.Message {
	messages := make([]*registryv1alpha1.Message, 0, messageDescriptors.Len())

	for i := 0; i < messageDescriptors.Len(); i++ {
		messageDescriptor := messageDescriptors.Get(i)
		message := g.GetMessage(messageDescriptor)
		messages = append(messages, message)
	}

	return messages
}

func (g *documentGeneratorImpl) GetMessage(messageDescriptor protoreflect.MessageDescriptor) *registryv1alpha1.Message {
	if message, ok := g.messageSet[string(messageDescriptor.FullName())]; ok {
		return message
	}

	// get location info
	messageLocation := messageDescriptor.ParentFile().SourceLocations().ByDescriptor(messageDescriptor)
	// get options
	messageOptions := protoutil.ProtoFromMessageDescriptor(messageDescriptor).Options

	message := &registryv1alpha1.Message{
		Name:        string(messageDescriptor.Name()),
		NestedName:  g.getNestedName(string(messageDescriptor.FullName()), string(messageDescriptor.ParentFile().Package())),
		FullName:    string(messageDescriptor.FullName()),
		Description: messageLocation.LeadingComments,
		FilePath:    messageDescriptor.ParentFile().Path(),
		IsMapEntry:  messageDescriptor.IsMapEntry(),
		Location:    g.toProtoLocation(messageLocation),
		// TODO MessageExtensions: nil,
		MessageOptions: &registryv1alpha1.MessageOptions{
			Deprecated: messageOptions.GetDeprecated(),
		},
		// TODO ImplicitlyDeprecated: false,
	}

	// fill message fields
	fields := g.GetFields(messageDescriptor.Fields())

	// fill message one of
	oneofs := g.GetOneofs(messageDescriptor.Oneofs())

	messageFields := make([]*registryv1alpha1.MessageField, 0, len(fields)+len(oneofs))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		messageFields = append(messageFields, &registryv1alpha1.MessageField{
			MessageField: &registryv1alpha1.MessageField_Field{
				Field: field,
			},
		})
	}
	for i := 0; i < len(oneofs); i++ {
		oneof := oneofs[i]
		messageFields = append(messageFields, &registryv1alpha1.MessageField{
			MessageField: &registryv1alpha1.MessageField_Oneof{
				Oneof: oneof,
			},
		})
	}
	message.Fields = messageFields

	// 记录message
	g.messageSet[string(messageDescriptor.FullName())] = message

	return message
}

func (g *documentGeneratorImpl) GetFields(fieldDescriptors protoreflect.FieldDescriptors) []*registryv1alpha1.Field {
	fields := make([]*registryv1alpha1.Field, 0, fieldDescriptors.Len())

	for i := 0; i < fieldDescriptors.Len(); i++ {
		fieldDescriptor := fieldDescriptors.Get(i)
		fields = append(fields, g.GetField(fieldDescriptor))
	}

	return fields
}

func (g *documentGeneratorImpl) GetField(fieldDescriptor protoreflect.FieldDescriptor) *registryv1alpha1.Field {
	fieldLocation := fieldDescriptor.ParentFile().SourceLocations().ByDescriptor(fieldDescriptor)
	fieldOptions := protoutil.ProtoFromFieldDescriptor(fieldDescriptor).GetOptions()

	field := &registryv1alpha1.Field{
		Name:        string(fieldDescriptor.Name()),
		Description: fieldLocation.LeadingComments,
		Label:       fieldDescriptor.Cardinality().String(),
		NestedType:  fieldDescriptor.Kind().String(),
		FullType:    fieldDescriptor.Kind().String(),
		Tag:         uint32(fieldDescriptor.Number()),
		// TODO Extendee:        "",
		FieldOptions: &registryv1alpha1.FieldOptions{
			Deprecated: fieldOptions.GetDeprecated(),
			Ctype:      int32(fieldOptions.GetCtype()),
			Jstype:     int32(fieldOptions.GetJstype()),
		},
	}
	if fieldOptions.GetPacked() {
		packed := true
		field.FieldOptions.Packed = &packed
	}

	// field kind is Message
	if fieldMessageDescriptor := fieldDescriptor.Message(); fieldMessageDescriptor != nil {
		field.FullType = string(fieldMessageDescriptor.FullName())
		field.NestedType = g.getNestedName(string(fieldMessageDescriptor.FullName()), string(fieldMessageDescriptor.ParentFile().Package()))
		field.ImportModuleRef = g.getImportModuleRef(fieldMessageDescriptor)
	}
	// field kind is Enum
	if fieldEnumDescriptor := fieldDescriptor.Enum(); fieldEnumDescriptor != nil {
		field.FullType = string(fieldEnumDescriptor.FullName())
		field.NestedType = g.getNestedName(string(fieldEnumDescriptor.FullName()), string(fieldEnumDescriptor.ParentFile().FullName()))
	}

	// field kind is MapEntry
	if fieldDescriptor.IsMap() {
		keyDescriptor := fieldDescriptor.MapKey()
		valueField := g.GetField(fieldDescriptor.MapValue())
		mapEntry := &registryv1alpha1.MapEntry{
			KeyFullType:          keyDescriptor.Kind().String(),
			ValueNestedType:      valueField.GetNestedType(),
			ValueFullType:        valueField.GetFullType(),
			ValueImportModuleRef: valueField.GetImportModuleRef(),
		}

		field.MapEntry = mapEntry
	}

	return field
}

func (g *documentGeneratorImpl) GetOneofs(oneofDescriptors protoreflect.OneofDescriptors) []*registryv1alpha1.Oneof {
	oneofs := make([]*registryv1alpha1.Oneof, 0, oneofDescriptors.Len())

	for i := 0; i < oneofDescriptors.Len(); i++ {
		oneofDescriptor := oneofDescriptors.Get(i)

		oneof := &registryv1alpha1.Oneof{
			Name:   string(oneofDescriptor.Name()),
			Fields: g.GetFields(oneofDescriptor.Fields()),
		}

		oneofs = append(oneofs, oneof)
	}

	return oneofs
}

func (g *documentGeneratorImpl) GetPackageEnums() []*registryv1alpha1.Enum {
	var enums []*registryv1alpha1.Enum

	for i := 0; i < len(g.packageLinkers); i++ {
		packageLink := g.packageLinkers[i]
		enums = append(enums, g.GetEnums(packageLink.Enums())...)
	}

	return enums
}

func (g *documentGeneratorImpl) GetEnums(enumDescriptors protoreflect.EnumDescriptors) []*registryv1alpha1.Enum {
	enums := make([]*registryv1alpha1.Enum, 0, enumDescriptors.Len())

	for i := 0; i < enumDescriptors.Len(); i++ {
		enumDescriptor := enumDescriptors.Get(i)
		enum := g.GetEnum(enumDescriptor)
		enums = append(enums, enum)
	}

	return enums
}

func (g *documentGeneratorImpl) GetEnum(enumDescriptor protoreflect.EnumDescriptor) *registryv1alpha1.Enum {
	// get location info
	enumLocation := enumDescriptor.ParentFile().SourceLocations().ByDescriptor(enumDescriptor)
	// get options
	enumOptions := protoutil.ProtoFromEnumDescriptor(enumDescriptor).GetOptions()

	enum := &registryv1alpha1.Enum{
		Name:        string(enumDescriptor.Name()),
		NestedName:  g.getNestedName(string(enumDescriptor.FullName()), string(enumDescriptor.ParentFile().Package())),
		FullName:    string(enumDescriptor.FullName()),
		Description: enumLocation.LeadingComments,
		FilePath:    enumDescriptor.ParentFile().Path(),
		Location:    g.toProtoLocation(enumLocation),
		EnumOptions: &registryv1alpha1.EnumOptions{
			Deprecated: enumOptions.GetDeprecated(),
			AllowAlias: enumOptions.GetAllowAlias(),
		},
		// TODO ImplicitlyDeprecated: false,
	}

	// enum values
	enumValues := g.GetEnumValues(enumDescriptor.Values())
	enum.Values = enumValues

	return enum
}

func (g *documentGeneratorImpl) GetEnumValues(enumValueDescriptor protoreflect.EnumValueDescriptors) []*registryv1alpha1.EnumValue {
	enumValues := make([]*registryv1alpha1.EnumValue, 0, enumValueDescriptor.Len())
	for i := 0; i < enumValueDescriptor.Len(); i++ {
		enumValueDescriptor := enumValueDescriptor.Get(i)
		enumValueLocation := enumValueDescriptor.ParentFile().SourceLocations().ByDescriptor(enumValueDescriptor)
		enumValueOptions := protoutil.ProtoFromEnumValueDescriptor(enumValueDescriptor).GetOptions()

		enumValue := &registryv1alpha1.EnumValue{
			Name:        string(enumValueDescriptor.Name()),
			Number:      int32(enumValueDescriptor.Number()),
			Description: enumValueLocation.LeadingComments,
			EnumValueOptions: &registryv1alpha1.EnumValueOptions{
				Deprecated: enumValueOptions.GetDeprecated(),
			},
		}

		enumValues = append(enumValues, enumValue)
	}

	return enumValues
}

func (g *documentGeneratorImpl) GetPackageServices() []*registryv1alpha1.Service {
	var services []*registryv1alpha1.Service

	for i := 0; i < len(g.packageLinkers); i++ {
		packageLink := g.packageLinkers[i]
		services = append(services, g.GetServices(packageLink.Services())...)
	}

	return services
}

func (g *documentGeneratorImpl) GetServices(serviceDescriptors protoreflect.ServiceDescriptors) []*registryv1alpha1.Service {
	services := make([]*registryv1alpha1.Service, 0, serviceDescriptors.Len())

	for i := 0; i < serviceDescriptors.Len(); i++ {
		serviceDescriptor := serviceDescriptors.Get(i)
		service := g.GetService(serviceDescriptor)
		services = append(services, service)
	}

	return services
}

func (g *documentGeneratorImpl) GetService(serviceDescriptor protoreflect.ServiceDescriptor) *registryv1alpha1.Service {
	// get location info
	serviceLocation := serviceDescriptor.ParentFile().SourceLocations().ByDescriptor(serviceDescriptor)
	// get options
	serviceOptions := protoutil.ProtoFromServiceDescriptor(serviceDescriptor).GetOptions()

	service := &registryv1alpha1.Service{
		Name:        string(serviceDescriptor.Name()),
		NestedName:  g.getNestedName(string(serviceDescriptor.FullName()), string(serviceDescriptor.ParentFile().Package())),
		FullName:    string(serviceDescriptor.FullName()),
		Description: serviceLocation.LeadingComments,
		FilePath:    serviceDescriptor.ParentFile().Path(),
		Location:    g.toProtoLocation(serviceLocation),
		ServiceOptions: &registryv1alpha1.ServiceOptions{
			Deprecated: serviceOptions.GetDeprecated(),
		},
		// TODO ImplicitlyDeprecated: false,
	}

	// methods
	methods := g.GetMethods(serviceDescriptor.Methods())
	service.Methods = methods

	return service
}

func (g *documentGeneratorImpl) GetMethods(methodDescriptors protoreflect.MethodDescriptors) []*registryv1alpha1.Method {
	methods := make([]*registryv1alpha1.Method, 0, methodDescriptors.Len())

	for i := 0; i < methodDescriptors.Len(); i++ {
		methodDescriptor := methodDescriptors.Get(i)
		methodLocation := methodDescriptor.ParentFile().SourceLocations().ByDescriptor(methodDescriptor)
		methodOptions := protoutil.ProtoFromMethodDescriptor(methodDescriptor).GetOptions()
		requestDescriptor := methodDescriptor.Input()
		responseDescriptor := methodDescriptor.Output()

		method := &registryv1alpha1.Method{
			Name:        string(methodDescriptor.Name()),
			Description: methodLocation.LeadingComments,
			Request:     g.getMethodRequestResponse(methodDescriptor.IsStreamingClient(), requestDescriptor),
			Response:    g.getMethodRequestResponse(methodDescriptor.IsStreamingServer(), responseDescriptor),
			MethodOptions: &registryv1alpha1.MethodOptions{
				Deprecated:       methodOptions.GetDeprecated(),
				IdempotencyLevel: int32(methodOptions.GetIdempotencyLevel()),
			},
			// TODO ImplicitlyDeprecated: false,
		}

		methods = append(methods, method)
	}

	return methods
}

func (g *documentGeneratorImpl) getMethodRequestResponse(streaming bool, descriptor protoreflect.MessageDescriptor) *registryv1alpha1.MethodRequestResponse {
	m := g.GetMessage(descriptor)

	r := &registryv1alpha1.MethodRequestResponse{
		NestedType: m.NestedName,
		FullType:   m.FullName,
		Streaming:  streaming,
		Message:    m,
	}

	r.ImportModuleRef = g.getImportModuleRef(descriptor)

	return r
}

func (g *documentGeneratorImpl) getImportModuleRef(descriptor protoreflect.Descriptor) *registryv1alpha1.ImportModuleRef {
	if g.isDependency(descriptor.ParentFile()) {
		// 如果是外部依赖
		fileDescriptor := descriptor.ParentFile()
		identity := g.parserAccessorHandler.ModuleIdentity(fileDescriptor.Path())
		importModuleRef := &registryv1alpha1.ImportModuleRef{
			Remote:      identity.Remote(),
			Owner:       identity.Owner(),
			Repository:  identity.Repository(),
			Commit:      g.parserAccessorHandler.Commit(fileDescriptor.Path()),
			PackageName: string(fileDescriptor.Package()),
		}

		return importModuleRef
	}

	return nil
}
