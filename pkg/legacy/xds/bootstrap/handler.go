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

package bootstrap

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
)

import (
	"github.com/go-logr/logr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	"github.com/apache/dubbo-kubernetes/pkg/xds/bootstrap/types"
)

var log = core.Log.WithName("bootstrap")

type BootstrapHandler struct {
	Generator BootstrapGenerator
}

func (b *BootstrapHandler) Handle(resp http.ResponseWriter, req *http.Request) {
	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.Error(err, "Could not read a request")
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	reqParams := types.BootstrapRequest{}
	if err := json.Unmarshal(bytes, &reqParams); err != nil {
		log.Error(err, "Could not parse a request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// The host doesn't have a port so we just use it directly
		hostname = host
	}

	reqParams.Host = hostname
	logger := log.WithValues("params", reqParams)

	config, dubboDpBootstrap, err := b.Generator.Generate(req.Context(), reqParams)
	if err != nil {
		handleError(resp, err, logger)
		return
	}

	bootstrapBytes, err := proto.ToYAML(config)
	if err != nil {
		logger.Error(err, "Could not convert to json")
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	var responseBytes []byte
	if req.Header.Get("accept") == "application/json" {
		resp.Header().Set("content-type", "application/json")
		response := createBootstrapResponse(bootstrapBytes, &dubboDpBootstrap)
		responseBytes, err = json.Marshal(response)
		if err != nil {
			logger.Error(err, "Could not convert to json")
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		// backwards compatibility
		resp.Header().Set("content-type", "text/x-yaml")
		responseBytes = bootstrapBytes
	}

	resp.WriteHeader(http.StatusOK)
	_, err = resp.Write(responseBytes)
	if err != nil {
		logger.Error(err, "Error while writing the response")
		return
	}
}

func handleError(resp http.ResponseWriter, err error, logger logr.Logger) {
	if err == DpTokenRequired || validators.IsValidationError(err) {
		resp.WriteHeader(http.StatusUnprocessableEntity)
		_, err = resp.Write([]byte(err.Error()))
		if err != nil {
			logger.Error(err, "Error while writing the response")
		}
		return
	}
	if ISSANMismatchErr(err) || err == NotCA {
		resp.WriteHeader(http.StatusBadRequest)
		if _, err := resp.Write([]byte(err.Error())); err != nil {
			logger.Error(err, "Error while writing the response")
		}
		return
	}
	if store.IsResourceNotFound(err) {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	logger.Error(err, "Could not generate a bootstrap configuration")
	resp.WriteHeader(http.StatusInternalServerError)
}

func createBootstrapResponse(bootstrap []byte, config *DubboDpBootstrap) *types.BootstrapResponse {
	bootstrapConfig := types.BootstrapResponse{
		Bootstrap: bootstrap,
	}
	aggregate := []types.Aggregate{}
	for _, value := range config.AggregateMetricsConfig {
		aggregate = append(aggregate, types.Aggregate{
			Address: value.Address,
			Name:    value.Name,
			Port:    value.Port,
			Path:    value.Path,
		})
	}
	bootstrapConfig.DubboSidecarConfiguration = types.DubboSidecarConfiguration{
		Metrics: types.MetricsConfiguration{
			Aggregate: aggregate,
		},
		Networking: types.NetworkingConfiguration{
			IsUsingTransparentProxy: config.NetworkingConfig.IsUsingTransparentProxy,
			Address:                 config.NetworkingConfig.Address,
			CorefileTemplate:        config.NetworkingConfig.CorefileTemplate,
		},
	}
	return &bootstrapConfig
}
