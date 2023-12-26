package mock

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	g "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rf "google.golang.org/grpc/reflection"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	pb "google.golang.org/grpc/reflection/grpc_testing"
	pbv3 "google.golang.org/grpc/reflection/grpc_testing_not_regenerate"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"net"
	"reflect"
	"regexp"
	"testing"
	"time"
)

const defaultTestTimeout = 10 * time.Second

var (
	romanRegex *regexp.Regexp
	// fileDescriptor of each  proto file.
	fdTest       *descriptorpb.FileDescriptorProto
	fdTestv3     *descriptorpb.FileDescriptorProto
	fdProto2     *descriptorpb.FileDescriptorProto
	fdProto2Ext  *descriptorpb.FileDescriptorProto
	fdProto2Ext2 *descriptorpb.FileDescriptorProto
	fdDynamic    *descriptorpb.FileDescriptorProto
	// reflection descriptors.
	fdDynamicFile protoreflect.FileDescriptor
	// fileDescriptor marshalled.
	fdTestByte       []byte
	fdTestv3Byte     []byte
	fdProto2Byte     []byte
	fdProto2ExtByte  []byte
	fdProto2Ext2Byte []byte
	fdDynamicByte    []byte
	mr               *MockRequest
)

func TestAtomic(t *testing.T) {
	assert.Equal(t, romanRegex, romanRegex)
}

func initProtoFile() {
	fdTest, fdTestByte = mr.LoadFileDesc("mock/proto/test.proto")
	fdTestv3, fdTestv3Byte = mr.LoadFileDesc("mock/proto/testv3.proto")
	fdProto2, fdProto2Byte = mr.LoadFileDesc("mock/proto/proto2.proto")
	fdProto2Ext, fdProto2ExtByte = mr.LoadFileDesc("mock/proto/proto2_ext.proto")
	mr = &MockRequest{}
	fdProto2Ext2, fdProto2Ext2Byte = mr.LoadFileDesc("mock/proto/proto2_ext2.proto")
	fdDynamic, fdDynamicFile, fdDynamicByte = mr.LoadFileDescDynamic(pbv3.FileDynamicProtoRawDesc)
}

func TestReflectionEnd2end(t *testing.T) {
	initProtoFile()
	// Start server.
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
	}
	//reflect server
	s := g.NewServer()
	pb.RegisterSearchServiceServer(s, &server{})
	pbv3.RegisterSearchServiceV3Server(s, &serverV3{})

	RegisterDynamicProto(s, fdDynamic, fdDynamicFile)

	// Register reflection service on s.
	rf.Register(s)
	go s.Serve(lis)

	// Create client.
	conn, err := g.Dial(lis.Addr().String(), g.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("cannot connect to server: %v", err)
	}
	defer conn.Close()

	c := rpb.NewServerReflectionClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	stream, err := c.ServerReflectionInfo(ctx, g.WaitForReady(true))
	if err != nil {
		fmt.Printf("cannot get ServerReflectionInfo: %v", err)
	}

	testListServices(t, stream)

	testFileByFilename(t, stream)

	s.Stop()
}

func testFileByFilename(t *testing.T, stream rpb.ServerReflection_ServerReflectionInfoClient) {
	for _, test := range []struct {
		filename string
		want     []byte
	}{
		{"mock/proto/test.proto", fdTestByte},
		{"mock/proto/proto2.proto", fdProto2Byte},
		{"mock/proto/proto2_ext.proto", fdProto2ExtByte},
	} {
		if err := stream.Send(&rpb.ServerReflectionRequest{
			MessageRequest: &rpb.ServerReflectionRequest_FileByFilename{
				FileByFilename: test.filename,
			},
		}); err != nil {
			t.Fatalf("failed to send request: %v", err)
		}
		r, err := stream.Recv()
		if err != nil {
			// io.EOF is not ok.
			t.Fatalf("failed to recv response: %v", err)
		}

		switch r.MessageResponse.(type) {
		case *rpb.ServerReflectionResponse_FileDescriptorResponse:
			if !reflect.DeepEqual(r.GetFileDescriptorResponse().FileDescriptorProto[0], test.want) {
				t.Errorf("FileByFilename(%v)\nreceived: %q,\nwant: %q", test.filename, r.GetFileDescriptorResponse().FileDescriptorProto[0], test.want)
			}
		default:
			t.Errorf("FileByFilename(%v) = %v, want type <ServerReflectionResponse_FileDescriptorResponse>", test.filename, r.MessageResponse)
		}
	}
}

func testListServices(t *testing.T, stream rpb.ServerReflection_ServerReflectionInfoClient) {

	services := mr.GetListServices(stream)
	want := []string{
		"mock.proto.SearchServiceV3",
		"mock.proto.SearchService",
	}

	m := make(map[string]int)
	for _, e := range services {
		m[e.Name]++
	}

	for _, e := range want {
		if m[e] > 0 {
			m[e]--
			continue
		}
		assert.Equal(t, want, services)
	}

}

type server struct {
	pb.UnimplementedSearchServiceServer
}

func (s *server) Search(ctx context.Context, in *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{}, nil
}

func (s *server) StreamingSearch(stream pb.SearchService_StreamingSearchServer) error {
	return nil
}

type serverV3 struct{}

func (s *serverV3) Search(ctx context.Context, in *pbv3.SearchRequestV3) (*pbv3.SearchResponseV3, error) {
	return &pbv3.SearchResponseV3{}, nil
}

func (s *serverV3) StreamingSearch(stream pbv3.SearchServiceV3_StreamingSearchServer) error {
	return nil
}
