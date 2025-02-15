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

package certs

import (
	"github.com/emicklei/go-restful/v3"

	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

func ClientCertAuthenticator(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	if user.FromCtx(request.Request.Context()).Name == user.Anonymous.Name && // do not overwrite existing user
		request.Request.TLS != nil &&
		request.Request.TLS.HandshakeComplete &&
		len(request.Request.TLS.PeerCertificates) > 0 {
		request.Request = request.Request.WithContext(user.Ctx(request.Request.Context(), user.Admin.Authenticated()))
	}
	chain.ProcessFilter(request, response)
}
