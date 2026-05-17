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

package wellknown

// Network filter names
const (
	// HTTPConnectionManager network filter
	HTTPConnectionManager = "filters.network.http_connection_manager"
)

// HTTP filter names
const (
	// JWTAuthentication validates JWT tokens when they are present.
	JWTAuthentication = "filters.http.jwt_authn"
	// HTTPRoleBasedAccessControl enforces request authorization policies.
	HTTPRoleBasedAccessControl = "filters.http.rbac"
	// HTTPRouter forwards the request after earlier HTTP filters have accepted it.
	HTTPRouter = "filters.http.router"
)
