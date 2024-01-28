package endpoints

import (
	"errors"

	"github.com/golang/protobuf/proto"

	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	endpoints_v3 "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/endpoints/v3"
)

func CreateClusterLoadAssignment(clusterName string, endpoints []core_xds.Endpoint, apiVersion core_xds.APIVersion) (proto.Message, error) {
	switch apiVersion {
	case core_xds.APIVersion(envoy_common.APIV3):
		return endpoints_v3.CreateClusterLoadAssignment(clusterName, endpoints), nil
	default:
		return nil, errors.New("unknown API")
	}
}
