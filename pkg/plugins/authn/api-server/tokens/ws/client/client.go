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

package client

import (
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/ws"
	"github.com/apache/dubbo-kubernetes/pkg/tokens"
	util_http "github.com/apache/dubbo-kubernetes/pkg/util/http"
)

type UserTokenClient interface {
	Generate(name string, groups []string, validFor time.Duration) (string, error)
}

var _ UserTokenClient = &httpUserTokenClient{}

func NewHTTPUserTokenClient(client util_http.Client) UserTokenClient {
	return &httpUserTokenClient{
		client: tokens.NewTokenClient(client, "user"),
	}
}

type httpUserTokenClient struct {
	client tokens.TokenClient
}

func (h *httpUserTokenClient) Generate(name string, groups []string, validFor time.Duration) (string, error) {
	tokenReq := &ws.UserTokenRequest{
		Name:     name,
		Groups:   groups,
		ValidFor: validFor.String(),
	}
	return h.client.Generate(tokenReq)
}
