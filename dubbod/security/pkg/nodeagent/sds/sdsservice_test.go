//
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

package sds

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	xdsmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	core "github.com/kdubbo/xds-api/core/v1"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
	discovery "github.com/kdubbo/xds-api/service/discovery/v1"
	"google.golang.org/grpc/metadata"
)

func TestDeltaSecretsRespondsToSubscribeAndPush(t *testing.T) {
	service := &sdsservice{
		st: &fakeSecretManager{
			secrets: map[string]*security.SecretItem{
				"default": {
					ResourceName:     "default",
					CertificateChain: []byte("cert-v1"),
					PrivateKey:       []byte("key-v1"),
				},
			},
		},
		stop:    make(chan struct{}),
		clients: map[string]*Context{},
	}
	stream := newFakeDeltaSecretsStream()
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.DeltaSecrets(stream)
	}()

	stream.recvCh <- &discovery.DeltaDiscoveryRequest{
		Node:                   &core.Node{Id: "proxyless~10.0.0.1~pod-1~app.svc.cluster.local"},
		TypeUrl:                xdsmodel.SecretType,
		ResourceNamesSubscribe: []string{"default"},
	}
	resp := stream.takeResponse(t)
	if resp.TypeUrl != xdsmodel.SecretType {
		t.Fatalf("TypeUrl = %s, want %s", resp.TypeUrl, xdsmodel.SecretType)
	}
	if resp.Nonce == "" || resp.SystemVersionInfo == "" {
		t.Fatalf("nonce/version = %q/%q, want both set", resp.Nonce, resp.SystemVersionInfo)
	}
	assertTLSSecret(t, resp, "default", []byte("cert-v1"), []byte("key-v1"))

	service.st.(*fakeSecretManager).secrets["default"] = &security.SecretItem{
		ResourceName:     "default",
		CertificateChain: []byte("cert-v2"),
		PrivateKey:       []byte("key-v2"),
	}
	service.push("default")
	pushResp := stream.takeResponse(t)
	if pushResp.TypeUrl != xdsmodel.SecretType {
		t.Fatalf("push TypeUrl = %s, want %s", pushResp.TypeUrl, xdsmodel.SecretType)
	}
	if pushResp.Nonce == resp.Nonce {
		t.Fatalf("push nonce = %s, want a new nonce", pushResp.Nonce)
	}
	assertTLSSecret(t, pushResp, "default", []byte("cert-v2"), []byte("key-v2"))

	close(stream.recvCh)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("DeltaSecrets() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("DeltaSecrets() did not exit after client close")
	}
}

func TestGenerateBuildsRootValidationContext(t *testing.T) {
	service := &sdsservice{
		st: &fakeSecretManager{
			secrets: map[string]*security.SecretItem{
				security.RootCertReqResourceName: {
					ResourceName: security.RootCertReqResourceName,
					RootCert:     []byte("root"),
				},
			},
		},
	}

	resp, err := service.generate([]string{security.RootCertReqResourceName})
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if resp.TypeUrl != xdsmodel.SecretType {
		t.Fatalf("TypeUrl = %s, want %s", resp.TypeUrl, xdsmodel.SecretType)
	}
	if len(resp.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resp.Resources))
	}
	secret := &tlsv1.Secret{}
	if err := resp.Resources[0].UnmarshalTo(secret); err != nil {
		t.Fatalf("unmarshal secret: %v", err)
	}
	if secret.GetName() != security.RootCertReqResourceName {
		t.Fatalf("secret name = %s, want %s", secret.GetName(), security.RootCertReqResourceName)
	}
	if got := secret.GetValidationContext().GetTrustedCa().GetInlineBytes(); !bytes.Equal(got, []byte("root")) {
		t.Fatalf("trusted CA = %q, want root", got)
	}
}

func TestFetchSecretsReturnsGeneratedResources(t *testing.T) {
	service := &sdsservice{
		st: &fakeSecretManager{
			secrets: map[string]*security.SecretItem{
				"default": {
					ResourceName:     "default",
					CertificateChain: []byte("cert"),
					PrivateKey:       []byte("key"),
				},
			},
		},
	}

	resp, err := service.FetchSecrets(context.Background(), &discovery.DiscoveryRequest{
		ResourceNames: []string{"default"},
	})
	if err != nil {
		t.Fatalf("FetchSecrets() error = %v", err)
	}
	if len(resp.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resp.Resources))
	}
	secret := &tlsv1.Secret{}
	if err := resp.Resources[0].UnmarshalTo(secret); err != nil {
		t.Fatalf("unmarshal secret: %v", err)
	}
	if got := secret.GetTlsCertificate().GetCertificateChain().GetInlineBytes(); !bytes.Equal(got, []byte("cert")) {
		t.Fatalf("cert = %q, want cert", got)
	}
}

func TestShouldRespondDeltaReportsNewSubscriptions(t *testing.T) {
	ctx := &Context{
		BaseConnection: xds.NewConnection("", nil),
		w:              &Watch{},
	}

	shouldRespond, _ := ctx.shouldRespondDelta(&discovery.DeltaDiscoveryRequest{
		TypeUrl:                xdsmodel.SecretType,
		ResourceNamesSubscribe: []string{"default"},
	})
	if !shouldRespond {
		t.Fatalf("initial subscribe should respond")
	}

	shouldRespond, delta := ctx.shouldRespondDelta(&discovery.DeltaDiscoveryRequest{
		TypeUrl:                xdsmodel.SecretType,
		ResourceNamesSubscribe: []string{"next"},
	})
	if !shouldRespond {
		t.Fatalf("new subscribe should respond")
	}
	if !delta.Subscribed.Contains("next") {
		t.Fatalf("Subscribed = %v, want next", delta.Subscribed.UnsortedList())
	}
	if delta.Subscribed.Contains("default") {
		t.Fatalf("Subscribed = %v, want only newly subscribed resources", delta.Subscribed.UnsortedList())
	}
}

func TestDeltaResourceNamesSkipsUnsubscribeOnlyResources(t *testing.T) {
	ctx := &Context{w: &Watch{}}
	ctx.w.NewWatchedResource(xdsmodel.SecretType, []string{"kept"})

	names := ctx.deltaResourceNames(xdsmodel.SecretType, xds.ResourceDelta{
		Unsubscribed: sets.New("removed"),
	})
	if len(names) != 0 {
		t.Fatalf("resource names = %v, want none for unsubscribe-only delta", names)
	}
}

type fakeSecretManager struct {
	secrets map[string]*security.SecretItem
}

func (f *fakeSecretManager) GenerateSecret(resourceName string) (*security.SecretItem, error) {
	return f.secrets[resourceName], nil
}

func assertTLSSecret(t *testing.T, resp *discovery.DeltaDiscoveryResponse, name string, cert, key []byte) {
	t.Helper()
	if len(resp.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resp.Resources))
	}
	if resp.Resources[0].GetName() != name {
		t.Fatalf("resource name = %s, want %s", resp.Resources[0].GetName(), name)
	}
	secret := &tlsv1.Secret{}
	if err := resp.Resources[0].GetResource().UnmarshalTo(secret); err != nil {
		t.Fatalf("unmarshal secret: %v", err)
	}
	if secret.GetName() != name {
		t.Fatalf("secret name = %s, want %s", secret.GetName(), name)
	}
	if got := secret.GetTlsCertificate().GetCertificateChain().GetInlineBytes(); !bytes.Equal(got, cert) {
		t.Fatalf("cert = %q, want %q", got, cert)
	}
	if got := secret.GetTlsCertificate().GetPrivateKey().GetInlineBytes(); !bytes.Equal(got, key) {
		t.Fatalf("key = %q, want %q", got, key)
	}
}

type fakeDeltaSecretsStream struct {
	ctx    context.Context
	recvCh chan *discovery.DeltaDiscoveryRequest
	sendCh chan *discovery.DeltaDiscoveryResponse
}

func newFakeDeltaSecretsStream() *fakeDeltaSecretsStream {
	return &fakeDeltaSecretsStream{
		ctx:    context.Background(),
		recvCh: make(chan *discovery.DeltaDiscoveryRequest, 8),
		sendCh: make(chan *discovery.DeltaDiscoveryResponse, 8),
	}
}

func (f *fakeDeltaSecretsStream) Send(resp *discovery.DeltaDiscoveryResponse) error {
	f.sendCh <- resp
	return nil
}

func (f *fakeDeltaSecretsStream) Recv() (*discovery.DeltaDiscoveryRequest, error) {
	req, ok := <-f.recvCh
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func (f *fakeDeltaSecretsStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *fakeDeltaSecretsStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *fakeDeltaSecretsStream) SetTrailer(metadata.MD) {}

func (f *fakeDeltaSecretsStream) Context() context.Context {
	return f.ctx
}

func (f *fakeDeltaSecretsStream) SendMsg(any) error {
	return nil
}

func (f *fakeDeltaSecretsStream) RecvMsg(any) error {
	return nil
}

func (f *fakeDeltaSecretsStream) takeResponse(t *testing.T) *discovery.DeltaDiscoveryResponse {
	t.Helper()
	select {
	case resp := <-f.sendCh:
		return resp
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delta SDS response")
		return nil
	}
}
