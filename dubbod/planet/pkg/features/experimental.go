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
	CACertConfigMapName = env.Register("PLANET_CA_CERT_CONFIGMAP", "dubbo-ca-root-cert",
		"The name of the ConfigMap that stores the Root CA Certificate that is used by dubbod").Get()
	EnableLeaderElection = env.Register("ENABLE_LEADER_ELECTION", true,
		"If enabled (default), starts a leader election client and gains leadership before executing controllers. "+
			"If false, it assumes that only one instance of istiod is running and skips leader election.").Get()
	EnableEnhancedSubsetRuleMerge = env.Register("ENABLE_ENHANCED_DESTINATIONRULE_MERGE", true,
		"If enabled, Dubbo merge subsetrules considering their exportTo fields,"+
			" they will be kept as independent rules if the exportTos are not equal.").Get()
)
