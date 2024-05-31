// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.20.0
// source: api/mesh/v1alpha1/mds.proto

package v1alpha1

import (
	reflect "reflect"
	sync "sync"
)

import (
	_ "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"

	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MappingRegisterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace       string   `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	ApplicationName string   `protobuf:"bytes,2,opt,name=applicationName,proto3" json:"applicationName,omitempty"`
	InterfaceNames  []string `protobuf:"bytes,3,rep,name=interfaceNames,proto3" json:"interfaceNames,omitempty"`
	PodName         string   `protobuf:"bytes,4,opt,name=podName,proto3" json:"podName,omitempty"`
}

func (x *MappingRegisterRequest) Reset() {
	*x = MappingRegisterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MappingRegisterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MappingRegisterRequest) ProtoMessage() {}

func (x *MappingRegisterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MappingRegisterRequest.ProtoReflect.Descriptor instead.
func (*MappingRegisterRequest) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{0}
}

func (x *MappingRegisterRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *MappingRegisterRequest) GetApplicationName() string {
	if x != nil {
		return x.ApplicationName
	}
	return ""
}

func (x *MappingRegisterRequest) GetInterfaceNames() []string {
	if x != nil {
		return x.InterfaceNames
	}
	return nil
}

func (x *MappingRegisterRequest) GetPodName() string {
	if x != nil {
		return x.PodName
	}
	return ""
}

type MappingRegisterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *MappingRegisterResponse) Reset() {
	*x = MappingRegisterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MappingRegisterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MappingRegisterResponse) ProtoMessage() {}

func (x *MappingRegisterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MappingRegisterResponse.ProtoReflect.Descriptor instead.
func (*MappingRegisterResponse) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{1}
}

func (x *MappingRegisterResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *MappingRegisterResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type MetaDataRegisterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string    `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	PodName   string    `protobuf:"bytes,2,opt,name=podName,proto3" json:"podName,omitempty"`   // dubbo的应用实例名, 由sdk通过环境变量获取
	Metadata  *MetaData `protobuf:"bytes,3,opt,name=metadata,proto3" json:"metadata,omitempty"` // 上报的元数据
}

func (x *MetaDataRegisterRequest) Reset() {
	*x = MetaDataRegisterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetaDataRegisterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetaDataRegisterRequest) ProtoMessage() {}

func (x *MetaDataRegisterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetaDataRegisterRequest.ProtoReflect.Descriptor instead.
func (*MetaDataRegisterRequest) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{2}
}

func (x *MetaDataRegisterRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *MetaDataRegisterRequest) GetPodName() string {
	if x != nil {
		return x.PodName
	}
	return ""
}

func (x *MetaDataRegisterRequest) GetMetadata() *MetaData {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type MetaDataRegisterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *MetaDataRegisterResponse) Reset() {
	*x = MetaDataRegisterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetaDataRegisterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetaDataRegisterResponse) ProtoMessage() {}

func (x *MetaDataRegisterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetaDataRegisterResponse.ProtoReflect.Descriptor instead.
func (*MetaDataRegisterResponse) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{3}
}

func (x *MetaDataRegisterResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *MetaDataRegisterResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type MappingSyncRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace     string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Nonce         string `protobuf:"bytes,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	InterfaceName string `protobuf:"bytes,3,opt,name=interfaceName,proto3" json:"interfaceName,omitempty"`
}

func (x *MappingSyncRequest) Reset() {
	*x = MappingSyncRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MappingSyncRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MappingSyncRequest) ProtoMessage() {}

func (x *MappingSyncRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MappingSyncRequest.ProtoReflect.Descriptor instead.
func (*MappingSyncRequest) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{4}
}

func (x *MappingSyncRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *MappingSyncRequest) GetNonce() string {
	if x != nil {
		return x.Nonce
	}
	return ""
}

func (x *MappingSyncRequest) GetInterfaceName() string {
	if x != nil {
		return x.InterfaceName
	}
	return ""
}

type MappingSyncResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce    string     `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Revision int64      `protobuf:"varint,2,opt,name=revision,proto3" json:"revision,omitempty"`
	Mappings []*Mapping `protobuf:"bytes,3,rep,name=mappings,proto3" json:"mappings,omitempty"`
}

func (x *MappingSyncResponse) Reset() {
	*x = MappingSyncResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MappingSyncResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MappingSyncResponse) ProtoMessage() {}

func (x *MappingSyncResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MappingSyncResponse.ProtoReflect.Descriptor instead.
func (*MappingSyncResponse) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{5}
}

func (x *MappingSyncResponse) GetNonce() string {
	if x != nil {
		return x.Nonce
	}
	return ""
}

func (x *MappingSyncResponse) GetRevision() int64 {
	if x != nil {
		return x.Revision
	}
	return 0
}

func (x *MappingSyncResponse) GetMappings() []*Mapping {
	if x != nil {
		return x.Mappings
	}
	return nil
}

// 可以根据应用名和版本号进行获取
type MetadataSyncRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace       string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Nonce           string `protobuf:"bytes,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	ApplicationName string `protobuf:"bytes,3,opt,name=applicationName,proto3" json:"applicationName,omitempty"`
	Revision        string `protobuf:"bytes,4,opt,name=revision,proto3" json:"revision,omitempty"`
}

func (x *MetadataSyncRequest) Reset() {
	*x = MetadataSyncRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetadataSyncRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetadataSyncRequest) ProtoMessage() {}

func (x *MetadataSyncRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetadataSyncRequest.ProtoReflect.Descriptor instead.
func (*MetadataSyncRequest) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{6}
}

func (x *MetadataSyncRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *MetadataSyncRequest) GetNonce() string {
	if x != nil {
		return x.Nonce
	}
	return ""
}

func (x *MetadataSyncRequest) GetApplicationName() string {
	if x != nil {
		return x.ApplicationName
	}
	return ""
}

func (x *MetadataSyncRequest) GetRevision() string {
	if x != nil {
		return x.Revision
	}
	return ""
}

type MetadataSyncResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce     string      `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Revision  int64       `protobuf:"varint,2,opt,name=revision,proto3" json:"revision,omitempty"`
	MetaDatum []*MetaData `protobuf:"bytes,3,rep,name=metaDatum,proto3" json:"metaDatum,omitempty"`
}

func (x *MetadataSyncResponse) Reset() {
	*x = MetadataSyncResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetadataSyncResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetadataSyncResponse) ProtoMessage() {}

func (x *MetadataSyncResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_mesh_v1alpha1_mds_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetadataSyncResponse.ProtoReflect.Descriptor instead.
func (*MetadataSyncResponse) Descriptor() ([]byte, []int) {
	return file_api_mesh_v1alpha1_mds_proto_rawDescGZIP(), []int{7}
}

func (x *MetadataSyncResponse) GetNonce() string {
	if x != nil {
		return x.Nonce
	}
	return ""
}

func (x *MetadataSyncResponse) GetRevision() int64 {
	if x != nil {
		return x.Revision
	}
	return 0
}

func (x *MetadataSyncResponse) GetMetaDatum() []*MetaData {
	if x != nil {
		return x.MetaDatum
	}
	return nil
}

var File_api_mesh_v1alpha1_mds_proto protoreflect.FileDescriptor

var file_api_mesh_v1alpha1_mds_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x65, 0x73, 0x68, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2f, 0x6d, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x64,
	0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x1a, 0x2a, 0x65, 0x6e, 0x76, 0x6f, 0x79, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x2f, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2f, 0x76, 0x33, 0x2f, 0x64,
	0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20,
	0x61, 0x70, 0x69, 0x2f, 0x6d, 0x65, 0x73, 0x68, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1f, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x65, 0x73, 0x68, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2f, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xa2, 0x01, 0x0a, 0x16, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09,
	0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x28, 0x0a, 0x0f, 0x61, 0x70,
	0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0f, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x26, 0x0a, 0x0e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x12, 0x18, 0x0a, 0x07,
	0x70, 0x6f, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70,
	0x6f, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x4d, 0x0a, 0x17, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e,
	0x67, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x8c, 0x01, 0x0a, 0x17, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61,
	0x74, 0x61, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x70, 0x6f, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x70, 0x6f, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x39, 0x0a, 0x08, 0x6d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x64, 0x75,
	0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x22, 0x4e, 0x0a, 0x18, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x22, 0x6e, 0x0a, 0x12, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x53,
	0x79, 0x6e, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61,
	0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e,
	0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x24,
	0x0a, 0x0d, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x4e, 0x61, 0x6d, 0x65, 0x22, 0x81, 0x01, 0x0a, 0x13, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05,
	0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x6f, 0x6e,
	0x63, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x38,
	0x0a, 0x08, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1c, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x08,
	0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x22, 0x8f, 0x01, 0x0a, 0x13, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6e,
	0x6f, 0x6e, 0x63, 0x65, 0x12, 0x28, 0x0a, 0x0f, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x61,
	0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1a,
	0x0a, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x85, 0x01, 0x0a, 0x14, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x76,
	0x69, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x72, 0x65, 0x76,
	0x69, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x3b, 0x0a, 0x09, 0x6d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74,
	0x75, 0x6d, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f,
	0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d,
	0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x52, 0x09, 0x6d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74,
	0x75, 0x6d, 0x32, 0xbe, 0x03, 0x0a, 0x0e, 0x4d, 0x44, 0x53, 0x53, 0x79, 0x6e, 0x63, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x6c, 0x0a, 0x0f, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12, 0x2b, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f,
	0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d,
	0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2c, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65,
	0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x61, 0x70, 0x70,
	0x69, 0x6e, 0x67, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x6f, 0x0a, 0x10, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12, 0x2c, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e,
	0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x65,
	0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2d, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65,
	0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x61,
	0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x67, 0x0a, 0x0c, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x53, 0x79, 0x6e, 0x63, 0x12, 0x28, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65, 0x73,
	0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x29,
	0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x53, 0x79, 0x6e,
	0x63, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x28, 0x01, 0x30, 0x01, 0x12, 0x64, 0x0a,
	0x0b, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x53, 0x79, 0x6e, 0x63, 0x12, 0x27, 0x2e, 0x64,
	0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65, 0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x6d, 0x65,
	0x73, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x61, 0x70, 0x70,
	0x69, 0x6e, 0x67, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x28,
	0x01, 0x30, 0x01, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x61, 0x70, 0x61, 0x63, 0x68, 0x65, 0x2f, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2d, 0x6b,
	0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x65,
	0x73, 0x68, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_api_mesh_v1alpha1_mds_proto_rawDescOnce sync.Once
	file_api_mesh_v1alpha1_mds_proto_rawDescData = file_api_mesh_v1alpha1_mds_proto_rawDesc
)

func file_api_mesh_v1alpha1_mds_proto_rawDescGZIP() []byte {
	file_api_mesh_v1alpha1_mds_proto_rawDescOnce.Do(func() {
		file_api_mesh_v1alpha1_mds_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_mesh_v1alpha1_mds_proto_rawDescData)
	})
	return file_api_mesh_v1alpha1_mds_proto_rawDescData
}

var file_api_mesh_v1alpha1_mds_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_api_mesh_v1alpha1_mds_proto_goTypes = []interface{}{
	(*MappingRegisterRequest)(nil),   // 0: dubbo.mesh.v1alpha1.MappingRegisterRequest
	(*MappingRegisterResponse)(nil),  // 1: dubbo.mesh.v1alpha1.MappingRegisterResponse
	(*MetaDataRegisterRequest)(nil),  // 2: dubbo.mesh.v1alpha1.MetaDataRegisterRequest
	(*MetaDataRegisterResponse)(nil), // 3: dubbo.mesh.v1alpha1.MetaDataRegisterResponse
	(*MappingSyncRequest)(nil),       // 4: dubbo.mesh.v1alpha1.MappingSyncRequest
	(*MappingSyncResponse)(nil),      // 5: dubbo.mesh.v1alpha1.MappingSyncResponse
	(*MetadataSyncRequest)(nil),      // 6: dubbo.mesh.v1alpha1.MetadataSyncRequest
	(*MetadataSyncResponse)(nil),     // 7: dubbo.mesh.v1alpha1.MetadataSyncResponse
	(*MetaData)(nil),                 // 8: dubbo.mesh.v1alpha1.MetaData
	(*Mapping)(nil),                  // 9: dubbo.mesh.v1alpha1.Mapping
}
var file_api_mesh_v1alpha1_mds_proto_depIdxs = []int32{
	8, // 0: dubbo.mesh.v1alpha1.MetaDataRegisterRequest.metadata:type_name -> dubbo.mesh.v1alpha1.MetaData
	9, // 1: dubbo.mesh.v1alpha1.MappingSyncResponse.mappings:type_name -> dubbo.mesh.v1alpha1.Mapping
	8, // 2: dubbo.mesh.v1alpha1.MetadataSyncResponse.metaDatum:type_name -> dubbo.mesh.v1alpha1.MetaData
	0, // 3: dubbo.mesh.v1alpha1.MDSSyncService.MappingRegister:input_type -> dubbo.mesh.v1alpha1.MappingRegisterRequest
	2, // 4: dubbo.mesh.v1alpha1.MDSSyncService.MetadataRegister:input_type -> dubbo.mesh.v1alpha1.MetaDataRegisterRequest
	6, // 5: dubbo.mesh.v1alpha1.MDSSyncService.MetadataSync:input_type -> dubbo.mesh.v1alpha1.MetadataSyncRequest
	4, // 6: dubbo.mesh.v1alpha1.MDSSyncService.MappingSync:input_type -> dubbo.mesh.v1alpha1.MappingSyncRequest
	1, // 7: dubbo.mesh.v1alpha1.MDSSyncService.MappingRegister:output_type -> dubbo.mesh.v1alpha1.MappingRegisterResponse
	3, // 8: dubbo.mesh.v1alpha1.MDSSyncService.MetadataRegister:output_type -> dubbo.mesh.v1alpha1.MetaDataRegisterResponse
	7, // 9: dubbo.mesh.v1alpha1.MDSSyncService.MetadataSync:output_type -> dubbo.mesh.v1alpha1.MetadataSyncResponse
	5, // 10: dubbo.mesh.v1alpha1.MDSSyncService.MappingSync:output_type -> dubbo.mesh.v1alpha1.MappingSyncResponse
	7, // [7:11] is the sub-list for method output_type
	3, // [3:7] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_api_mesh_v1alpha1_mds_proto_init() }
func file_api_mesh_v1alpha1_mds_proto_init() {
	if File_api_mesh_v1alpha1_mds_proto != nil {
		return
	}
	file_api_mesh_v1alpha1_metadata_proto_init()
	file_api_mesh_v1alpha1_mapping_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_api_mesh_v1alpha1_mds_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MappingRegisterRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MappingRegisterResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetaDataRegisterRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetaDataRegisterResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MappingSyncRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MappingSyncResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetadataSyncRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_mesh_v1alpha1_mds_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetadataSyncResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_mesh_v1alpha1_mds_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_mesh_v1alpha1_mds_proto_goTypes,
		DependencyIndexes: file_api_mesh_v1alpha1_mds_proto_depIdxs,
		MessageInfos:      file_api_mesh_v1alpha1_mds_proto_msgTypes,
	}.Build()
	File_api_mesh_v1alpha1_mds_proto = out.File
	file_api_mesh_v1alpha1_mds_proto_rawDesc = nil
	file_api_mesh_v1alpha1_mds_proto_goTypes = nil
	file_api_mesh_v1alpha1_mds_proto_depIdxs = nil
}