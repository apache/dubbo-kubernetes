package stream

import (
	"context"
	"crypto/tls"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	url2 "net/url"
)

type Client struct {
	conn   *grpc.ClientConn
	client mesh_proto.MDSSyncServiceClient
}

type mappingStream struct {
	mappingStream  mesh_proto.MDSSyncService_MappingSyncClient
	latestACKed    *mesh_proto.MappingSyncResponse
	latestReceived *mesh_proto.MappingSyncResponse
}

type metadataStream struct {
	metadataStream mesh_proto.MDSSyncService_MetadataSyncClient
	latestACKed    *mesh_proto.MetadataSyncResponse
	latestReceived *mesh_proto.MetadataSyncResponse
}

func New(serverURL string) (*Client, error) {
	url, err := url2.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	var dialOpts []grpc.DialOption
	switch url.Scheme {
	case "grpc":
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	case "grpcs":
		// #nosec G402 -- it's acceptable as this is only to be used in testing
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})))
	default:
		return nil, errors.Errorf("unsupported scheme %q. Use one of %s", url.Scheme, []string{"grpc", "grpcs"})
	}
	conn, err := grpc.Dial(url.Host, dialOpts...)
	if err != nil {
		return nil, err
	}
	client := mesh_proto.NewMDSSyncServiceClient(conn)
	return &Client{
		conn:   conn,
		client: client,
	}, nil
}

func (c *Client) MappingRegister() {

}

func (c *Client) MetadataRegister() {

}

func (c *Client) StartMappingStream() (*mappingStream, error) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.MD{})
	stream, err := c.client.MappingSync(ctx)
	if err != nil {
		return nil, err
	}
	return &mappingStream{
		mappingStream:  stream,
		latestACKed:    &mesh_proto.MappingSyncResponse{},
		latestReceived: &mesh_proto.MappingSyncResponse{},
	}, nil
}

func (c *Client) StartMetadataSteam() (*metadataStream, error) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.MD{})
	stream, err := c.client.MetadataSync(ctx)
	if err != nil {
		return nil, err
	}
	return &metadataStream{
		metadataStream: stream,
		latestACKed:    &mesh_proto.MetadataSyncResponse{},
		latestReceived: &mesh_proto.MetadataSyncResponse{},
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (s *metadataStream) MetadataSyncRequest() {

}

func (s *metadataStream) WaitForMetadataResource() (*mesh_proto.MetadataSyncResponse, error) {
	resp, err := s.metadataStream.Recv()
	if err != nil {
		return nil, err
	}
	s.latestReceived = resp
	return resp, err
}

func (s *mappingStream) MappingSyncRequest() {

}

func (s *mappingStream) WaitForMappingResource() (*mesh_proto.MappingSyncResponse, error) {
	resp, err := s.mappingStream.Recv()
	if err != nil {
		return nil, err
	}
	s.latestReceived = resp
	return resp, err
}

func (s *mappingStream) Close() error {
	return s.mappingStream.CloseSend()
}

func (s *metadataStream) Close() error {
	return s.metadataStream.CloseSend()
}

func (s *mappingStream) MappingACK() error {
	latestReceived := s.latestReceived
	if latestReceived == nil {
		return nil
	}
	err := s.mappingStream.Send(&mesh_proto.MappingSyncRequest{
		Nonce: latestReceived.Nonce,
	})
	if err == nil {
		s.latestACKed = s.latestReceived
	}
	return err
}

func (s *metadataStream) MetadataACK() error {
	latestReceived := s.latestReceived
	if latestReceived == nil {
		return nil
	}
	err := s.metadataStream.Send(&mesh_proto.MetadataSyncRequest{
		Nonce: latestReceived.Nonce,
	})
	if err == nil {
		s.latestACKed = s.latestReceived
	}
	return err
}
