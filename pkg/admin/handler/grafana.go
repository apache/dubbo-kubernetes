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

package handler

// reverse proxy for grafana
import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func Grafana(rt core_runtime.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		baseUrl := rt.Config().Admin.Grafana
		grafanaURL := baseUrl + "grafana" + c.Param("any")
		proxyUrl, _ := url.Parse(grafanaURL)
		director := func(req *http.Request) {
			req.URL.Scheme = proxyUrl.Scheme
			req.URL.Host = proxyUrl.Host
			req.Host = proxyUrl.Host
			req.URL.Path = proxyUrl.Path
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
