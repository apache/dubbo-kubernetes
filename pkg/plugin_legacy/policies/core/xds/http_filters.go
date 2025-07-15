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

package xds

import (
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"github.com/pkg/errors"
)

func InsertHTTPFiltersBeforeRouter(manager *envoy_hcm.HttpConnectionManager, newFilters ...*envoy_hcm.HttpFilter) error {
	for i, filter := range manager.HttpFilters {
		if filter.Name == "envoy.filters.http.router" {
			// insert new filters before router
			manager.HttpFilters = append(append(manager.HttpFilters[:i:i], newFilters...), manager.HttpFilters[i:]...)
			return nil
		}
	}
	return errors.New("could not insert filter, envoy.filters.http.router is not found in HTTPConnectionManager")
}
