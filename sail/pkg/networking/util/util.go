package util

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	dubbonetworking "github.com/apache/dubbo-kubernetes/sail/pkg/networking"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"strings"
)

const (
	// PassthroughFilterChain to catch traffic that doesn't match other filter chains.
	PassthroughFilterChain = "PassthroughFilterChain"
)

func DelimitedStatsPrefix(statPrefix string) string {
	statPrefix += constants.StatPrefixDelimiter
	return statPrefix
}

func BuildAdditionalAddresses(extrAddresses []string, listenPort uint32) []*listener.AdditionalAddress {
	var additionalAddresses []*listener.AdditionalAddress
	if len(extrAddresses) > 0 {
		for _, exbd := range extrAddresses {
			if exbd == "" {
				continue
			}
			extraAddress := &listener.AdditionalAddress{
				Address: BuildAddress(exbd, listenPort),
			}
			additionalAddresses = append(additionalAddresses, extraAddress)
		}
	}
	return additionalAddresses
}

func BuildAddress(bind string, port uint32) *core.Address {
	address := BuildNetworkAddress(bind, port, dubbonetworking.TransportProtocolTCP)
	if address != nil {
		return address
	}

	return &core.Address{
		Address: &core.Address_Pipe{
			Pipe: &core.Pipe{
				Path: strings.TrimPrefix(bind, model.UnixAddressPrefix),
			},
		},
	}
}

func BuildNetworkAddress(bind string, port uint32, transport dubbonetworking.TransportProtocol) *core.Address {
	if port == 0 {
		return nil
	}
	return &core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Address:  bind,
				Protocol: transport.ToEnvoySocketProtocol(),
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

// ByteCount returns a human readable byte format
// Inspired by https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCount(b int) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
