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

package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

import (
	"github.com/asaskevich/govalidator"

	"github.com/golang-jwt/jwt/v4"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/envoy"
	"github.com/apache/dubbo-kubernetes/pkg/config/app/dubboctl"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	leader_memory "github.com/apache/dubbo-kubernetes/pkg/plugins/leader/memory"
	util_files "github.com/apache/dubbo-kubernetes/pkg/util/files"
)

type ProxyConfig struct {
	ComponentManager         component.Manager
	BootstrapDynamicMetadata map[string]string
	Config                   *dubboctl.Config
	BootstrapGenerator       envoy.BootstrapConfigFactoryFunc
	DataplaneTokenGenerator  func(cfg *dubboctl.Config) (component.Component, error)
}

var features = []string{core_xds.FeatureTCPAccessLogViaNamedPipe}

func defaultDataplaneTokenGenerator(cfg *dubboctl.Config) (component.Component, error) {
	if cfg.DataplaneRuntime.Token != "" {
		path := filepath.Join(cfg.DataplaneRuntime.ConfigDir, cfg.Dataplane.Name)
		if err := writeFile(path, []byte(cfg.DataplaneRuntime.Token), 0o600); err != nil {
			runLog.Error(err, "unable to create file with dataplane token")
			return nil, err
		}
		cfg.DataplaneRuntime.TokenPath = path
	}

	if cfg.DataplaneRuntime.TokenPath != "" {
		if err := ValidateTokenPath(cfg.DataplaneRuntime.TokenPath); err != nil {
			return nil, errors.Wrapf(err, "dataplane token is invalid, in Kubernetes you must mount a serviceAccount token, in universal you must start your proxy with a generated token.")
		}
	}

	return component.ComponentFunc(func(<-chan struct{}) error {
		return nil
	}), nil
}

func DefaultProxyConfig() *ProxyConfig {
	config := dubboctl.DefaultConfig()
	return &ProxyConfig{
		ComponentManager:         component.NewManager(leader_memory.NewNeverLeaderElector()),
		BootstrapGenerator:       envoy.NewRemoteBootstrapGenerator(runtime.GOOS, features),
		Config:                   &config,
		BootstrapDynamicMetadata: map[string]string{},
		DataplaneTokenGenerator:  defaultDataplaneTokenGenerator,
	}
}

func ValidateTokenPath(path string) error {
	if path == "" {
		return nil
	}
	empty, err := util_files.FileEmpty(path)
	if err != nil {
		return errors.Wrapf(err, "could not read file %s", path)
	}
	if empty {
		return errors.Errorf("token under file %s is empty", path)
	}

	rawToken, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "could not read the token in the file %s", path)
	}

	strToken := strings.TrimSpace(string(rawToken))
	if !govalidator.Matches(strToken, "^[^\\x00\\n\\r]*$") {
		return errors.New("Token shouldn't contain line breaks within the token, only at the start or end")
	}
	token, _, err := new(jwt.Parser).ParseUnverified(strToken, &jwt.MapClaims{})
	if err != nil {
		return errors.Wrap(err, "not valid JWT token. Can't parse it.")
	}

	if token.Method.Alg() == "" {
		return errors.New("not valid JWT token. No Alg.")
	}

	if token.Header == nil {
		return errors.New("not valid JWT token. No Header.")
	}

	return nil
}
