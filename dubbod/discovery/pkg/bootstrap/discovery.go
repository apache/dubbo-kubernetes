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

package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/apigen"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/core"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/networking/grpcgen"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
)

func InitGenerators(
	s *xds.DiscoveryServer,
	cg core.ConfigGenerator,
) {
	env := s.Env
	generators := map[string]model.XdsResourceGenerator{}
	edsGen := &xds.EdsGenerator{Cache: s.Cache, EndpointIndex: env.EndpointIndex}
	generators[v1.ClusterType] = &xds.CdsGenerator{ConfigGenerator: cg}
	generators[v1.ListenerType] = &xds.LdsGenerator{ConfigGenerator: cg}
	generators[v1.RouteType] = &xds.RdsGenerator{ConfigGenerator: cg}
	generators[v1.EndpointType] = edsGen

	generators["grpc"] = &grpcgen.GrpcConfigGenerator{}
	generators["grpc/"+v1.EndpointType] = edsGen
	generators["grpc/"+v1.ListenerType] = generators["grpc"]
	generators["grpc/"+v1.RouteType] = generators["grpc"]
	generators["grpc/"+v1.ClusterType] = generators["grpc"]

	generators["api"] = apigen.NewGenerator(env.ConfigStore)
	generators["api/"+v1.EndpointType] = edsGen

	generators["event"] = xds.NewStatusGen(s)
	s.Generators = generators
}
