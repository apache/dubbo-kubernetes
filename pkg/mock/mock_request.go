package mock

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type MockRequest struct{}

func (m *MockRequest) LoadFileDesc(filename string) (*descriptorpb.FileDescriptorProto, []byte) {
	fd, err := protoregistry.GlobalFiles.FindFileByPath(filename)
	if err != nil {
		panic(err)
	}

	fdProto := protodesc.ToFileDescriptorProto(fd)
	b, err := proto.Marshal(fdProto)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal fd: %v", err))
	}
	return fdProto, b
}

func (m *MockRequest) LoadFileDescDynamic(b []byte) (*descriptorpb.FileDescriptorProto, protoreflect.FileDescriptor, []byte) {
	fileDescriptorProto := new(descriptorpb.FileDescriptorProto)
	if err := proto.Unmarshal(b, fileDescriptorProto); err != nil {
		panic(fmt.Sprintf("failed to unmarshal dynamic proto raw descriptor"))
	}

	fd, err := protodesc.NewFile(fileDescriptorProto, nil)
	if err != nil {
		panic(err)
	}

	err = protoregistry.GlobalFiles.RegisterFile(fd)
	if err != nil {
		panic(err)
	}

	for i := 0; i < fd.Messages().Len(); i++ {
		fileDescriptorProto := fd.Messages().Get(i)
		if err := protoregistry.GlobalTypes.RegisterMessage(dynamicpb.NewMessageType(fileDescriptorProto)); err != nil {
			panic(err)
		}
	}

	return fileDescriptorProto, fd, b
}

func (m *MockRequest) GetListServices(stream rpb.ServerReflection_ServerReflectionInfoClient) []*rpb.ServiceResponse {
	if err := stream.Send(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{},
	}); err != nil {
		fmt.Printf("failed to send request: %v", err)
	}
	r, err := stream.Recv()
	if err != nil {
		// io.EOF is not ok.
		fmt.Printf("failed to recv response: %v", err)
	}

	var services []*rpb.ServiceResponse

	switch r.MessageResponse.(type) {
	case *rpb.ServerReflectionResponse_ListServicesResponse:
		services = r.GetListServicesResponse().Service

	default:
		fmt.Printf("ListServices = %v, want type <ServerReflectionResponse_ListServicesResponse>", r.MessageResponse)
	}
	return services
}

func RegisterDynamicProto(srv *grpc.Server, fdp *descriptorpb.FileDescriptorProto, fd protoreflect.FileDescriptor) {
	type emptyInterface interface{}

	for i := 0; i < fd.Services().Len(); i++ {
		s := fd.Services().Get(i)

		sd := &grpc.ServiceDesc{
			ServiceName: string(s.FullName()),
			HandlerType: (*emptyInterface)(nil),
			Metadata:    fdp.GetName(),
		}

		for j := 0; j < s.Methods().Len(); j++ {
			m := s.Methods().Get(j)
			sd.Methods = append(sd.Methods, grpc.MethodDesc{
				MethodName: string(m.Name()),
			})
		}

		srv.RegisterService(sd, struct{}{})
	}
}
