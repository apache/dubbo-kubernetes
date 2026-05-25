package multicluster

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
)

const EastWestGatewayEnvName = "DUBBO_EASTWEST_GATEWAYS"

type EastWestGateway struct {
	Cluster cluster.ID
	Address string
	Port    uint32
}

func (g EastWestGateway) Endpoint() string {
	return net.JoinHostPort(g.Address, strconv.Itoa(int(g.Port)))
}

func ParseEastWestGateways(raw string) (map[cluster.ID]EastWestGateway, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	out := map[cluster.ID]EastWestGateway{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid east-west gateway entry %q, expected cluster=host:port", item)
		}
		clusterID := cluster.ID(strings.TrimSpace(parts[0]))
		host, portText, err := net.SplitHostPort(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid east-west gateway endpoint %q: %w", parts[1], err)
		}
		port, err := strconv.ParseUint(portText, 10, 32)
		if err != nil || port == 0 || port > 65535 {
			return nil, fmt.Errorf("invalid east-west gateway port %q", portText)
		}
		out[clusterID] = EastWestGateway{
			Cluster: clusterID,
			Address: host,
			Port:    uint32(port),
		}
	}
	return out, nil
}
