package stream

import (
	"context"
	"crypto/tls"
	url2 "net/url"

	"github.com/apache/dubbo-kubernetes/api/legacy"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	conn   *grpc.ClientConn
	client legacy.MDSSyncServiceClient
}

type mappingStream struct {
	mappingStream  legacy.MDSSyncService_MappingSyncClient
	latestACKed    *legacy.MappingSyncResponse
	latestReceived *legacy.MappingSyncResponse
}

type metadataStream struct {
	metadataStream legacy.MDSSyncService_MetadataSyncClient
	latestACKed    *legacy.MetadataSyncResponse
	latestReceived *legacy.MetadataSyncResponse
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
	client := legacy.NewMDSSyncServiceClient(conn)
	return &Client{
		conn:   conn,
		client: client,
	}, nil
}

func (c *Client) MappingRegister(ctx context.Context, req *legacy.MappingRegisterRequest) error {
	_, err := c.client.MappingRegister(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) MetadataRegister(ctx context.Context, req *legacy.MetaDataRegisterRequest) error {
	_, err := c.client.MetadataRegister(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) StartMappingStream() (*mappingStream, error) {
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.MD{})
	stream, err := c.client.MappingSync(ctx)
	if err != nil {
		return nil, err
	}
	return &mappingStream{
		mappingStream:  stream,
		latestACKed:    &legacy.MappingSyncResponse{},
		latestReceived: &legacy.MappingSyncResponse{},
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
		latestACKed:    &legacy.MetadataSyncResponse{},
		latestReceived: &legacy.MetadataSyncResponse{},
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (s *metadataStream) MetadataSyncRequest(req *legacy.MetadataSyncRequest) error {
	err := s.metadataStream.Send(req)
	if err != nil {
		return err
	}
	return nil
}

func (s *metadataStream) WaitForMetadataResource() (*legacy.MetadataSyncResponse, error) {
	resp, err := s.metadataStream.Recv()
	if err != nil {
		return nil, err
	}
	s.latestReceived = resp
	return resp, err
}

func (s *mappingStream) MappingSyncRequest(req *legacy.MappingSyncRequest) error {
	err := s.mappingStream.Send(req)
	if err != nil {
		return err
	}
	return nil
}

func (s *mappingStream) WaitForMappingResource() (*legacy.MappingSyncResponse, error) {
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
	err := s.mappingStream.Send(&legacy.MappingSyncRequest{
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
	err := s.metadataStream.Send(&legacy.MetadataSyncRequest{
		Nonce: latestReceived.Nonce,
	})
	if err == nil {
		s.latestACKed = s.latestReceived
	}
	return err
}
