package multizone

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

type DdsServerConfig struct {
	config.BaseConfig

	// Port of a gRPC server that serves Kuma Discovery Service (KDS).
	GrpcPort uint32 `json:"grpcPort" envconfig:"dubbo_multizone_global_kds_grpc_port"`
	// Interval for refreshing state of the world
	RefreshInterval config_types.Duration `json:"refreshInterval" envconfig:"dubbo_multizone_global_kds_refresh_interval"`
}
