/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package aegis

import (
	"context"
	"errors"
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/log"

	dubbogrpc "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/caclient"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	pb "istio.io/api/security/v1alpha1"
)

var aegisClientLog = log.RegisterScope("aegisclient", "aegis client debugging")

type TLSOptions struct {
	RootCert string
	Key      string
	Cert     string
}

type AegisClient struct {
	// It means enable tls connection to Citadel if this is not nil.
	tlsOpts  *TLSOptions
	client   pb.IstioCertificateServiceClient
	conn     *grpc.ClientConn
	provider credentials.PerRPCCredentials
	opts     *security.Options
}

func NewAegisClient(opts *security.Options, tlsOpts *TLSOptions) (*AegisClient, error) {
	c := &AegisClient{
		tlsOpts:  tlsOpts,
		opts:     opts,
		provider: caclient.NewDefaultTokenProvider(opts),
	}

	conn, err := c.buildConnection()
	if err != nil {
		aegisClientLog.Errorf("Failed to connect to endpoint %s: %v", opts.CAEndpoint, err)
		return nil, fmt.Errorf("failed to connect to endpoint %s", opts.CAEndpoint)
	}
	c.conn = conn
	c.client = pb.NewIstioCertificateServiceClient(conn)
	return c, nil
}

func (c *AegisClient) CSRSign(csrPEM []byte, certValidTTLInSec int64) (res []string, err error) {
	crMetaStruct := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			security.CertSigner: {
				Kind: &structpb.Value_StringValue{StringValue: c.opts.CertSigner},
			},
		},
	}
	req := &pb.IstioCertificateRequest{
		Csr:              string(csrPEM),
		ValidityDuration: certValidTTLInSec,
		Metadata:         crMetaStruct,
	}

	defer func() {
		if err != nil {
			aegisClientLog.Errorf("failed to sign CSR: %v", err)
			if err := c.reconnect(); err != nil {
				aegisClientLog.Errorf("failed reconnect: %v", err)
			}
		}
	}()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("ClusterID", c.opts.ClusterID))
	for k, v := range c.opts.CAHeaders {
		ctx = metadata.AppendToOutgoingContext(ctx, k, v)
	}

	resp, err := c.client.CreateCertificate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %v", err)
	}

	if len(resp.CertChain) <= 1 {
		return nil, errors.New("invalid empty CertChain")
	}

	return resp.CertChain, nil
}

func (c *AegisClient) GetRootCertBundle() ([]string, error) {
	return []string{}, nil
}

func (c *AegisClient) getTLSOptions() *dubbogrpc.TLSOptions {
	if c.tlsOpts != nil {
		return &dubbogrpc.TLSOptions{
			RootCert:      c.tlsOpts.RootCert,
			Key:           c.tlsOpts.Key,
			Cert:          c.tlsOpts.Cert,
			ServerAddress: c.opts.CAEndpoint,
			SAN:           c.opts.CAEndpointSAN,
		}
	}
	return nil
}

func (c *AegisClient) buildConnection() (*grpc.ClientConn, error) {
	tlsOpts := c.getTLSOptions()
	opts, err := dubbogrpc.ClientOptions(nil, tlsOpts)
	if err != nil {
		return nil, err
	}
	opts = append(opts,
		grpc.WithPerRPCCredentials(c.provider),
		security.CARetryInterceptor(),
	)
	conn, err := grpc.Dial(c.opts.CAEndpoint, opts...)
	if err != nil {
		aegisClientLog.Errorf("Failed to connect to endpoint %s: %v", c.opts.CAEndpoint, err)
		return nil, fmt.Errorf("failed to connect to endpoint %s", c.opts.CAEndpoint)
	}

	return conn, nil
}

func (c *AegisClient) reconnect() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %v", err)
	}

	conn, err := c.buildConnection()
	if err != nil {
		return err
	}
	c.conn = conn
	c.client = pb.NewIstioCertificateServiceClient(conn)
	aegisClientLog.Info("recreated connection")
	return nil
}

func (c *AegisClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
