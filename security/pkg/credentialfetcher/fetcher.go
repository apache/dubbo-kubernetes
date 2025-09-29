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


package credentialfetcher

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/credentialfetcher/plugin"
)

func NewCredFetcher(credtype, trustdomain, jwtPath, identityProvider string) (security.CredFetcher, error) {
	switch credtype {
	case security.JWT, "":
		// If unset, also default to JWT for backwards compatibility
		if jwtPath == "" {
			return nil, nil // no cred fetcher - using certificates only
		}
		return plugin.CreateTokenPlugin(jwtPath), nil
	default:
		return nil, fmt.Errorf("invalid credential fetcher type %s", credtype)
	}
}
