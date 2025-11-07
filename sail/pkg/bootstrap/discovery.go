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

package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/apigen"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/core"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/grpcgen"
	"github.com/apache/dubbo-kubernetes/sail/pkg/xds"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
)

func InitGenerators(
	s *xds.DiscoveryServer,
	cg core.ConfigGenerator,
) {
	env := s.Env
	generators := map[string]model.XdsResourceGenerator{}
	edsGen := &xds.EdsGenerator{Cache: s.Cache, EndpointIndex: env.EndpointIndex}
	generators[v3.ClusterType] = &xds.CdsGenerator{ConfigGenerator: cg}
	generators[v3.ListenerType] = &xds.LdsGenerator{ConfigGenerator: cg}
	generators[v3.RouteType] = &xds.RdsGenerator{ConfigGenerator: cg}
	generators[v3.EndpointType] = edsGen

	generators["grpc"] = &grpcgen.GrpcConfigGenerator{}
	generators["grpc/"+v3.EndpointType] = edsGen
	generators["grpc/"+v3.ListenerType] = generators["grpc"]
	generators["grpc/"+v3.RouteType] = generators["grpc"]
	generators["grpc/"+v3.ClusterType] = generators["grpc"]

	generators["api"] = apigen.NewGenerator(env.ConfigStore)
	generators["api/"+v3.EndpointType] = edsGen

	generators["event"] = xds.NewStatusGen(s)
	s.Generators = generators
}
