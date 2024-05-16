package api

import (
	"context"
	"fmt"
	greetgrpc "github.com/apache/dubbo-kubernetes/pkg/admin/util/reflection/testdata/proto/grpc_gen"
	"io"
	"strings"

	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"

	greet "github.com/apache/dubbo-kubernetes/pkg/admin/util/reflection/testdata/proto"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util/reflection/testdata/proto/triple_gen/greettriple"
)

type GreetGRPCServer struct {
	greetgrpc.UnimplementedGreetServiceServer
}

func (src *GreetGRPCServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
	resp := &greet.GreetResponse{Greeting: req.Name}
	return resp, nil
}

func (src *GreetGRPCServer) GreetStream(stream greetgrpc.GreetService_GreetStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if triple.IsEnded(err) {
				break
			}
			return fmt.Errorf("triple BidiStream recv error: %s", err)
		}
		if err := stream.Send(&greet.GreetStreamResponse{Greeting: req.Name}); err != nil {
			return fmt.Errorf("triple BidiStream send error: %s", err)
		}
	}
	return nil
}

func (src *GreetGRPCServer) GreetClientStream(stream greetgrpc.GreetService_GreetClientStreamServer) error {
	var reqs []string
	for {

		req := greet.GreetClientStreamRequest{}
		err := stream.RecvMsg(&req)
		if err != nil && err != io.EOF {
			return fmt.Errorf("triple ClientStream recv err: %s", err)
		}
		if err == io.EOF {
			break
		}

		reqs = append(reqs, req.Name)
	}

	resp := &greet.GreetClientStreamResponse{
		Greeting: strings.Join(reqs, ","),
	}

	return stream.SendMsg(resp)
}

func (src *GreetGRPCServer) GreetServerStream(req *greet.GreetServerStreamRequest, stream greetgrpc.GreetService_GreetServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&greet.GreetServerStreamResponse{Greeting: req.Name}); err != nil {
			return fmt.Errorf("triple ServerStream send err: %s", err)
		}
	}
	return nil
}

type GreetTripleServer struct {
}

func (srv *GreetTripleServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
	resp := &greet.GreetResponse{Greeting: req.Name}
	return resp, nil
}

func (srv *GreetTripleServer) GreetStream(ctx context.Context, stream greettriple.GreetService_GreetStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if triple.IsEnded(err) {
				break
			}
			return fmt.Errorf("triple BidiStream recv error: %s", err)
		}
		if err := stream.Send(&greet.GreetStreamResponse{Greeting: req.Name}); err != nil {
			return fmt.Errorf("triple BidiStream send error: %s", err)
		}
	}
	return nil
}

func (srv *GreetTripleServer) GreetClientStream(ctx context.Context, stream greettriple.GreetService_GreetClientStreamServer) (*greet.GreetClientStreamResponse, error) {
	var reqs []string
	for stream.Recv() {
		reqs = append(reqs, stream.Msg().Name)
	}
	if stream.Err() != nil && !triple.IsEnded(stream.Err()) {
		return nil, fmt.Errorf("triple ClientStream recv err: %s", stream.Err())
	}
	resp := &greet.GreetClientStreamResponse{
		Greeting: strings.Join(reqs, ","),
	}

	return resp, nil
}

func (srv *GreetTripleServer) GreetServerStream(ctx context.Context, req *greet.GreetServerStreamRequest, stream greettriple.GreetService_GreetServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&greet.GreetServerStreamResponse{Greeting: req.Name}); err != nil {
			return fmt.Errorf("triple ServerStream send err: %s", err)
		}
	}
	return nil
}
