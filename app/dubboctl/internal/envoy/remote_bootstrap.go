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

package envoy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	net_url "net/url"
	"os"
	"strings"
	"time"
)

import (
	envoy_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"

	"github.com/pkg/errors"

	"github.com/sethvargo/go-retry"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/app/dubboctl"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	dubbo_version "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/apache/dubbo-kubernetes/pkg/xds/bootstrap/types"
)

var (
	log           = core.Log.WithName("dataplane")
	DpNotFoundErr = errors.New("Dataplane entity not found. If you are running on Universal please create a Dataplane entity on Dubbo-cp before starting Dubbo-dp or pass it to Dubbo-dp run --dataplane-file=/file. If you are running on Kubernetes, please check the Dubbo-cp logs to determine why the Dataplane entity could not be created by the automatic sidecar injection.")
)

type remoteBootstrap struct {
	operatingSystem string
	features        []string
}

func InvalidRequestErr(msg string) error {
	return errors.Errorf("Invalid request: %s", msg)
}

func IsInvalidRequestErr(err error) bool {
	return strings.HasPrefix(err.Error(), "Invalid request: ")
}

func NewRemoteBootstrapGenerator(operatingSystem string, features []string) BootstrapConfigFactoryFunc {
	rb := remoteBootstrap{
		operatingSystem: operatingSystem,
		features:        features,
	}
	return rb.Generate
}

func (b *remoteBootstrap) Generate(ctx context.Context, url string, cfg dubboctl.Config, params BootstrapParams) (*envoy_bootstrap_v3.Bootstrap, *types.DubboSidecarConfiguration, error) {
	bootstrapUrl, err := net_url.Parse(url)
	if err != nil {
		return nil, nil, err
	}
	client := &http.Client{Timeout: time.Second * 10}

	if bootstrapUrl.Scheme == "https" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		if cfg.ControlPlane.CaCert != "" {
			certPool := x509.NewCertPool()
			if ok := certPool.AppendCertsFromPEM([]byte(cfg.ControlPlane.CaCert)); !ok {
				return nil, nil, errors.New("could not add certificate")
			}
			tlsConfig.RootCAs = certPool
		} else {
			log.Info(`[WARNING] The data plane proxy cannot verify the identity of the control plane because you are not setting the "--ca-cert-file" argument or setting the DUBBO_CONTROL_PLANE_CA_CERT environment variable.`)
			tlsConfig.InsecureSkipVerify = true // #nosec G402 -- we have the warning above
		}
	}

	backoff := retry.WithMaxDuration(cfg.ControlPlane.Retry.MaxDuration.Duration, retry.NewConstant(cfg.ControlPlane.Retry.Backoff.Duration))
	var respBytes []byte
	err = retry.Do(ctx, backoff, func(ctx context.Context) error {
		log.Info("trying to fetch bootstrap configuration from the Control Plane")
		bootstrapUrl.Path = "/bootstrap"
		respBytes, err = b.requestForBootstrap(ctx, client, bootstrapUrl, cfg, params)
		if err == nil {
			return nil
		}
		if IsInvalidRequestErr(err) { // there is no point in retrying invalid request
			return err
		}

		switch err {
		case DpNotFoundErr:
			log.Info("Dataplane entity is not yet found in the Control Plane. If you are running on Kubernetes, the control plane is most likely still in the process of converting Pod to Dataplane. If it takes too long, check pod events and control plane logs to see possible cause. Retrying.", "backoff", cfg.ControlPlane.Retry.Backoff)
		default:
			log.Info("could not fetch bootstrap configuration, make sure you are not trying to connect to global-cp. retrying (this could help only if you're connecting to zone-cp).", "backoff", cfg.ControlPlane.Retry.Backoff, "err", err.Error())
		}
		return retry.RetryableError(err)
	})
	if err != nil {
		return nil, nil, err
	}

	bootstrap := &types.BootstrapResponse{}
	if err := json.Unmarshal(respBytes, bootstrap); err != nil {
		return nil, nil, err
	}

	envoyBootstrap := &envoy_bootstrap_v3.Bootstrap{}
	if err := util_proto.FromYAML(bootstrap.Bootstrap, envoyBootstrap); err != nil {
		return nil, nil, err
	}
	return envoyBootstrap, &bootstrap.DubboSidecarConfiguration, nil
}

func (b *remoteBootstrap) resourceMetadata(cfg dubboctl.DataplaneResources) types.ProxyResources {
	var maxMemory uint64

	if cfg.MaxMemoryBytes == 0 {
		maxMemory = DetectMaxMemory()
	} else {
		maxMemory = cfg.MaxMemoryBytes
	}

	res := types.ProxyResources{}

	if maxMemory != 0 {
		res.MaxHeapSizeBytes = maxMemory
	}

	return res
}

func (b *remoteBootstrap) requestForBootstrap(ctx context.Context, client *http.Client, url *net_url.URL, cfg dubboctl.Config, params BootstrapParams) ([]byte, error) {
	url.Scheme = "http"
	var dataplaneResource string
	if params.Dataplane != nil {
		dpJSON, err := json.Marshal(params.Dataplane)
		if err != nil {
			return nil, err
		}
		dataplaneResource = string(dpJSON)
	}
	token := ""
	if cfg.DataplaneRuntime.TokenPath != "" {
		tokenData, err := os.ReadFile(cfg.DataplaneRuntime.TokenPath)
		if err != nil {
			return nil, err
		}
		token = string(tokenData)
	}
	if cfg.DataplaneRuntime.Token != "" {
		token = cfg.DataplaneRuntime.Token
	}
	// Remove any trailing and starting spaces.
	token = strings.TrimSpace(token)

	resources := b.resourceMetadata(cfg.DataplaneRuntime.Resources)

	request := types.BootstrapRequest{
		Mesh:               cfg.Dataplane.Mesh,
		Name:               cfg.Dataplane.Name,
		ProxyType:          cfg.Dataplane.ProxyType,
		DataplaneToken:     token,
		DataplaneTokenPath: cfg.DataplaneRuntime.TokenPath,
		DataplaneResource:  dataplaneResource,
		CaCert:             cfg.ControlPlane.CaCert,
		Version: types.Version{
			DubboDp: types.DubboDpVersion{
				Version:   dubbo_version.Build.Version,
				GitTag:    dubbo_version.Build.GitTag,
				GitCommit: dubbo_version.Build.GitCommit,
				BuildDate: dubbo_version.Build.BuildDate,
			},
			Envoy: types.EnvoyVersion{
				Version:           params.EnvoyVersion.Version,
				Build:             params.EnvoyVersion.Build,
				DubboDpCompatible: params.EnvoyVersion.DubboDpCompatible,
			},
		},
		DynamicMetadata:     params.DynamicMetadata,
		DNSPort:             params.DNSPort,
		EmptyDNSPort:        params.EmptyDNSPort,
		OperatingSystem:     b.operatingSystem,
		Features:            b.features,
		Resources:           resources,
		Workdir:             params.Workdir,
		AccessLogSocketPath: params.AccessLogSocketPath,
		MetricsResources: types.MetricsResources{
			SocketPath: params.MetricsSocketPath,
			CertPath:   params.MetricsCertPath,
			KeyPath:    params.MetricsKeyPath,
		},
	}
	jsonBytes, err := json.MarshalIndent(request, "", " ")
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request to json")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request to bootstrap server failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to read the response with status code: %d. Make sure you are using https URL", resp.StatusCode)
		}
		if resp.StatusCode == http.StatusNotFound && len(bodyBytes) == 0 {
			return nil, DpNotFoundErr
		}
		if resp.StatusCode == http.StatusNotFound && string(bodyBytes) == "404: Page Not Found" { // response body of Go HTTP Server when hit for invalid endpoint
			return nil, errors.New("There is no /bootstrap endpoint for provided CP address. Double check if the address passed to the CP has a DP Server port (5678 by default), not HTTP API (5681 by default)")
		}
		if resp.StatusCode/100 == 4 {
			return nil, InvalidRequestErr(string(bodyBytes))
		}
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read the body of the response")
	}
	return respBytes, nil
}
