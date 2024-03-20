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

package admin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

import (
	envoy_admin_v3 "github.com/envoyproxy/go-control-plane/envoy/admin/v3"

	"github.com/pkg/errors"
)

import (
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

type EnvoyAdminClient interface {
	PostQuit(ctx context.Context, dataplane *core_mesh.DataplaneResource) error

	Stats(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error)
	Clusters(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error)
	ConfigDump(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error)
}

type envoyAdminClient struct {
	rm               manager.ResourceManager
	defaultAdminPort uint32

	caCertPool *x509.CertPool
	clientCert *tls.Certificate
}

func NewEnvoyAdminClient(rm manager.ResourceManager, adminPort uint32) EnvoyAdminClient {
	client := &envoyAdminClient{
		rm:               rm,
		defaultAdminPort: adminPort,
	}
	return client
}

func (a *envoyAdminClient) buildHTTPClient(ctx context.Context) (*http.Client, error) {
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 3 * time.Second,
			}).DialContext,
		},
		Timeout: 5 * time.Second,
	}
	return c, nil
}

const (
	quitquitquit = "quitquitquit"
)

func (a *envoyAdminClient) PostQuit(ctx context.Context, dataplane *core_mesh.DataplaneResource) error {
	httpClient, err := a.buildHTTPClient(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/%s", dataplane.AdminAddress(a.defaultAdminPort), quitquitquit)
	request, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	// Envoy will not send back any response, so do we not check the response
	response, err := httpClient.Do(request)
	if errors.Is(err, io.EOF) {
		return nil // Envoy may not respond correctly for this request because it already started the shut-down process.
	}
	if err != nil {
		return errors.Wrapf(err, "unable to send POST to %s", quitquitquit)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("envoy response [%d %s] [%s]", response.StatusCode, response.Status, response.Body)
	}

	return nil
}

func (a *envoyAdminClient) Stats(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	return a.executeRequest(ctx, proxy, "stats")
}

func (a *envoyAdminClient) Clusters(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	return a.executeRequest(ctx, proxy, "clusters")
}

func (a *envoyAdminClient) ConfigDump(ctx context.Context, proxy core_model.ResourceWithAddress) ([]byte, error) {
	configDump, err := a.executeRequest(ctx, proxy, "config_dump")
	if err != nil {
		return nil, err
	}

	cd := &envoy_admin_v3.ConfigDump{}
	if err := util_proto.FromJSON(configDump, cd); err != nil {
		return nil, err
	}

	if err := Sanitize(cd); err != nil {
		return nil, err
	}

	return util_proto.ToJSONIndent(cd, " ")
}

func (a *envoyAdminClient) executeRequest(ctx context.Context, proxy core_model.ResourceWithAddress, path string) ([]byte, error) {
	var httpClient *http.Client
	var err error
	u := &url.URL{}

	switch proxy.(type) {
	case *core_mesh.DataplaneResource:
		httpClient, err = a.buildHTTPClient(ctx)
		if err != nil {
			return nil, err
		}
		u.Scheme = "https"
	case *core_mesh.ZoneIngressResource, *core_mesh.ZoneEgressResource:
		httpClient, err = a.buildHTTPClient(ctx)
		if err != nil {
			return nil, err
		}
		u.Scheme = "https"
	default:
		return nil, errors.New("unsupported proxy type")
	}

	if host, _, err := net.SplitHostPort(proxy.AdminAddress(a.defaultAdminPort)); err == nil && host == "127.0.0.1" {
		httpClient = &http.Client{
			Timeout: 5 * time.Second,
		}
		u.Scheme = "http"
	}

	u.Host = proxy.AdminAddress(a.defaultAdminPort)
	u.Path = path
	request, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to send GET to %s", "config_dump")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("envoy response [%d %s] [%s]", response.StatusCode, response.Status, response.Body)
	}

	resp, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
