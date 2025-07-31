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

package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	EnableXDSCaching = env.Register("NAVIGATOR_ENABLE_XDS_CACHE", true,
		"If true, Navigator will cache XDS responses.").Get()

	// EnableCDSCaching determines if CDS caching is enabled. This is explicitly split out of ENABLE_XDS_CACHE,
	// so that in case there are issues with the CDS cache we can just disable the CDS cache.
	EnableCDSCaching = env.Register("PILOT_ENABLE_CDS_CACHE", true,
		"If true, Pilot will cache CDS responses. Note: this depends on PILOT_ENABLE_XDS_CACHE.").Get()

	// EnableRDSCaching determines if RDS caching is enabled. This is explicitly split out of ENABLE_XDS_CACHE,
	// so that in case there are issues with the RDS cache we can just disable the RDS cache.
	EnableRDSCaching = env.Register("PILOT_ENABLE_RDS_CACHE", true,
		"If true, Pilot will cache RDS responses. Note: this depends on PILOT_ENABLE_XDS_CACHE.").Get()
)
