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

package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	EnableCDSCaching = env.Register("PLANET_ENABLE_CDS_CACHE", true,
		"If true, PLANET will cache CDS responses. Note: this depends on PLANET_ENABLE_XDS_CACHE.").Get()

	EnableRDSCaching = env.Register("PLANET_ENABLE_RDS_CACHE", true,
		"If true, PLANET will cache RDS responses. Note: this depends on PLANET_ENABLE_XDS_CACHE.").Get()

	EnableXDSCaching = env.Register("PLANET_ENABLE_XDS_CACHE", true,
		"If true, Planet will cache XDS responses.").Get()
)
