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

package caclient

import (
	"context"
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/security"
	"google.golang.org/grpc/credentials"
)

type DefaultTokenProvider struct {
	opts *security.Options
}

func NewDefaultTokenProvider(opts *security.Options) credentials.PerRPCCredentials {
	return &DefaultTokenProvider{opts}
}

func (t *DefaultTokenProvider) GetToken() (string, error) {
	if t.opts.CredFetcher == nil {
		return "", nil
	}
	token, err := t.opts.CredFetcher.GetPlatformCredential()
	if err != nil {
		return "", fmt.Errorf("fetch platform credential: %v", err)
	}

	return token, nil
}

func (t *DefaultTokenProvider) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	if t == nil {
		return nil, nil
	}
	token, err := t.GetToken()
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, nil
	}
	return map[string]string{
		"authorization": "Bearer " + token,
	}, nil
}

// Allow the token provider to be used regardless of transport security; callers can determine whether
// this is safe themselves.
func (t *DefaultTokenProvider) RequireTransportSecurity() bool {
	return false
}
