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

package gateway

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	k8sv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func GetStatus[I, IS any](spec I) IS {
	switch t := any(spec).(type) {
	case *k8sv1.HTTPRoute:
		return any(t.Status).(IS)
	case *k8sv1.Gateway:
		return any(t.Status).(IS)
	case *k8sv1.GatewayClass:
		return any(t.Status).(IS)
	default:
		log.Fatalf("unknown type %T", t)
		return ptr.Empty[IS]()
	}
}
