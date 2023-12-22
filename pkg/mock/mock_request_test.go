package mock

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rf "google.golang.org/grpc/reflection"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	pb "google.golang.org/grpc/reflection/grpc_testing"
	pbv3 "google.golang.org/grpc/reflection/grpc_testing_not_regenerate"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"net"
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
)

func TestAtomic(t *testing.T) {
	assert.Equal(t, romanRegex, romanRegex)
}

func initProtoFile() {
	romanRegex = regexp.MustCompile(`^M{0,3}(CM|CD|D?C{0,3})(XC|XL|L?X{0,3})(IX|IV|V?I{0,3})$`)
	fdTest, fdTestByte = LoadFileDesc("app/dubboctl/internal/mock/proto/test.proto")
	fdTestv3, fdTestv3Byte = LoadFileDesc("app/dubboctl/internal/mock/proto/testv3.proto")
	fdProto2, fdProto2Byte = LoadFileDesc("app/dubboctl/internal/mock/proto/proto2.proto")
	fdProto2Ext, fdProto2ExtByte = LoadFileDesc("app/dubboctl/internal/mock/proto/proto2_ext.proto")
	fdProto2Ext2, fdProto2Ext2Byte = LoadFileDesc("app/dubboctl/internal/mock/proto/proto2_ext2.proto")
	fdDynamic, fdDynamicFile, fdDynamicByte = LoadFileDescDynamic(pbv3.FileDynamicProtoRawDesc)
}

func TestReflectionEnd2end(t *testing.T) {
	initProtoFile()
	// Start server.
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
	}
	//reflect server
	s := grpc.NewServer()
	pb.RegisterSearchServiceServer(s, &server{})
	pbv3.RegisterSearchServiceV3Server(s, &serverV3{})

	RegisterDynamicProto(s, fdDynamic, fdDynamicFile)

	// Register reflection service on s.
	rf.Register(s)
	go s.Serve(lis)

	// Create client.
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("cannot connect to server: %v", err)
	}
	defer conn.Close()

	c := rpb.NewServerReflectionClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()
	stream, err := c.ServerReflectionInfo(ctx, grpc.WaitForReady(true))
	if err != nil {
		fmt.Printf("cannot get ServerReflectionInfo: %v", err)
	}

	testListServices(t, stream)

	s.Stop()
}

func testListServices(t *testing.T, stream rpb.ServerReflection_ServerReflectionInfoClient) {

	services := GetListServices(stream)
	want := []string{
		"grpc.testingv3.SearchServiceV3",
		"grpc.testing.SearchService",
		"grpc.reflection.v1alpha.ServerReflection",
		"grpc.testing.DynamicService",
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
