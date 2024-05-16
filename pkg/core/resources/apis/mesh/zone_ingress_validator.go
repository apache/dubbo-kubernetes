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

func (r *ZoneIngressResource) Validate() error {
	var err validators.ValidationError
	err.Add(r.validateNetworking(validators.RootedAt("networking"), r.Spec.GetNetworking()))
	err.Add(r.validateAvailableServices(validators.RootedAt("availableService"), r.Spec.GetAvailableServices()))
	return err.OrNil()
}

func (r *ZoneIngressResource) validateNetworking(path validators.PathBuilder, networking *mesh_proto.ZoneIngress_Networking) validators.ValidationError {
	var err validators.ValidationError
	if admin := networking.GetAdmin(); admin != nil {
		if r.UsesInboundInterface(IPv4Loopback, admin.GetPort()) {
			err.AddViolationAt(path.Field("admin").Field("port"), "must differ from port")
		}
	}
	if networking.GetAdvertisedAddress() == "" && networking.GetAdvertisedPort() != 0 {
		err.AddViolationAt(path.Field("advertisedAddress"), `has to be defined with advertisedPort`)
	}
	if networking.GetAdvertisedPort() == 0 && networking.GetAdvertisedAddress() != "" {
		err.AddViolationAt(path.Field("advertisedPort"), `has to be defined with advertisedAddress`)
	}
	if networking.GetAddress() != "" {
		err.Add(validateAddress(path.Field("address"), networking.GetAddress()))
	}
	if networking.GetAdvertisedAddress() != "" {
		err.Add(validateAddress(path.Field("advertisedAddress"), networking.GetAdvertisedAddress()))
	}

	err.Add(ValidatePort(path.Field("port"), networking.GetPort()))

	if networking.GetAdvertisedPort() != 0 {
		err.Add(ValidatePort(path.Field("advertisedPort"), networking.GetAdvertisedPort()))
	}

	return err
}

func (r *ZoneIngressResource) validateAvailableServices(path validators.PathBuilder, availableServices []*mesh_proto.ZoneIngress_AvailableService) validators.ValidationError {
	var err validators.ValidationError
	for i, availableService := range availableServices {
		p := path.Index(i)
		err.Add(ValidateTags(p.Field("tags"), availableService.Tags, ValidateTagsOpts{
			RequireService: true,
		}))
	}
	return err
}
