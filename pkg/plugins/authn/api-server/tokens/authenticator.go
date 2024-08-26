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

package tokens

import (
	"strings"

	"github.com/emicklei/go-restful/v3"

	"github.com/apache/dubbo-kubernetes/pkg/api-server/authn"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	rest_errors "github.com/apache/dubbo-kubernetes/pkg/core/rest/errors"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/issuer"
)

const bearerPrefix = "Bearer "

var log = core.Log.WithName("plugins").WithName("authn").WithName("api-server").WithName("tokens")

func UserTokenAuthenticator(validator issuer.UserTokenValidator) authn.Authenticator {
	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		if authn.SkipAuth(request) {
			chain.ProcessFilter(request, response)
			return
		}
		authnHeader := request.Request.Header.Get("authorization")
		if user.FromCtx(request.Request.Context()).Name == user.Anonymous.Name && // do not overwrite existing user
			authnHeader != "" &&
			strings.HasPrefix(authnHeader, bearerPrefix) {
			token := strings.TrimPrefix(authnHeader, bearerPrefix)
			u, err := validator.Validate(request.Request.Context(), token)
			if err != nil {
				rest_errors.HandleError(request.Request.Context(), response, &rest_errors.Unauthenticated{}, "Invalid authentication data")
				log.Info("authentication rejected", "reason", err.Error())
				return
			}
			request.Request = request.Request.WithContext(user.Ctx(request.Request.Context(), u.Authenticated()))
		}
		chain.ProcessFilter(request, response)
	}
}
