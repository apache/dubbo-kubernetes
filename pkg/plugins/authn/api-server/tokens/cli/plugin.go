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

package cli

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api"
	util_http "github.com/apache/dubbo-kubernetes/pkg/util/http"
)

const (
	AuthType = "tokens"
	TokenKey = "token"
)

type TokenAuthnPlugin struct{}

var _ api.AuthnPlugin = &TokenAuthnPlugin{}

func (t *TokenAuthnPlugin) Validate(authConf map[string]string) error {
	if authConf[TokenKey] == "" {
		return errors.New("provide token=YOUR_TOKEN")
	}
	return nil
}

func (t *TokenAuthnPlugin) DecorateClient(delegate util_http.Client, authConf map[string]string) (util_http.Client, error) {
	return util_http.ClientFunc(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("authorization", "Bearer "+authConf[TokenKey])
		return delegate.Do(req)
	}), nil
}
