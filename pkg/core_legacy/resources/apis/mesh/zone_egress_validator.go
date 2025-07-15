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

package mesh

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

func (r *ZoneEgressResource) Validate() error {
	var err validators.ValidationError
	err.Add(r.validateNetworking(validators.RootedAt("networking"), r.Spec.GetNetworking()))
	return err.OrNil()
}

func (r *ZoneEgressResource) validateNetworking(path validators.PathBuilder, networking *mesh_proto.ZoneEgress_Networking) validators.ValidationError {
	var err validators.ValidationError
	if admin := networking.GetAdmin(); admin != nil {
		if r.UsesInboundInterface(IPv4Loopback, admin.GetPort()) {
			err.AddViolationAt(path.Field("admin").Field("port"), "must differ from port")
		}
	}

	if networking.GetAddress() != "" {
		err.Add(validateAddress(path.Field("address"), networking.GetAddress()))
	}

	err.Add(ValidatePort(path.Field("port"), networking.GetPort()))

	return err
}
