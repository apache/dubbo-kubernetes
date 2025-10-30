package bootstrap

import (
	"net/http"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
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
	systemNameSpace string,
	clusterID cluster.ID,
	internalDebugMux *http.ServeMux,
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
