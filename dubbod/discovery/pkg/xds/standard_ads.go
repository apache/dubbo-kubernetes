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

package xds

import (
	"fmt"
	"sync"

	core "github.com/dubml/xds-api/core/v1"
	discovery "github.com/dubml/xds-api/service/discovery/v1"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	standardcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	standardtls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	standard "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	standardmatcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
)

// Standard SDKs share the existing authenticated ADS lifecycle and generators.
type standardADS struct {
	standard.UnimplementedAggregatedDiscoveryServiceServer
	server *DiscoveryServer
}

func (s *standardADS) StreamAggregatedResources(stream standard.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	return s.server.StreamAggregatedResources(newStandardStream(stream))
}

type standardStream struct {
	standard.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	state   sync.Mutex
	writer  sync.Mutex
	watched map[string]bool
	named   map[string]bool
	pending map[string]*standard.DiscoveryResponse
}

func newStandardStream(stream standard.AggregatedDiscoveryService_StreamAggregatedResourcesServer) *standardStream {
	return &standardStream{AggregatedDiscoveryService_StreamAggregatedResourcesServer: stream, watched: make(map[string]bool), named: make(map[string]bool), pending: make(map[string]*standard.DiscoveryResponse)}
}

var standardNames = map[string]string{
	"listener.v1.Listener":              "envoy.config.listener.v3.Listener",
	"route.v1.RouteConfiguration":       "envoy.config.route.v3.RouteConfiguration",
	"cluster.v1.Cluster":                "envoy.config.cluster.v3.Cluster",
	"endpoint.v1.ClusterLoadAssignment": "envoy.config.endpoint.v3.ClusterLoadAssignment",
	"extensions.filters.network.http_connection_manager.v1.HttpConnectionManager": "envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
	"extensions.filters.http.router.v1.Router":                                    "envoy.extensions.filters.http.router.v3.Router",
	"extensions.filters.http.rbac.v1.RBAC":                                        "envoy.extensions.filters.http.rbac.v3.RBAC",
	"extensions.filters.http.jwt_authn.v1.JwtAuthentication":                      "envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
	"extensions.transport_sockets.tls.v1.UpstreamTlsContext":                      "envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
	"extensions.transport_sockets.tls.v1.DownstreamTlsContext":                    "envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
}

func standardURL(custom string) (string, error) {
	const prefix = "type.googleapis.com/"
	for name, translated := range standardNames {
		if custom == prefix+name {
			return prefix + translated, nil
		}
	}
	return "", fmt.Errorf("unsupported SDK resource type %q", custom)
}

func (s *standardStream) Recv() (*discovery.DiscoveryRequest, error) {
	request, err := s.AggregatedDiscoveryService_StreamAggregatedResourcesServer.Recv()
	if err != nil {
		return nil, err
	}
	var customType string
	for name, translated := range standardNames {
		if request.TypeUrl == "type.googleapis.com/"+translated {
			customType = "type.googleapis.com/" + name
			break
		}
	}
	if customType == "" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported SDK subscription %q", request.TypeUrl)
	}
	converted := &discovery.DiscoveryRequest{VersionInfo: request.VersionInfo, ResourceNames: append([]string(nil), request.ResourceNames...), TypeUrl: customType, ResponseNonce: request.ResponseNonce, ErrorDetail: request.ErrorDetail}
	if node := request.Node; node != nil {
		converted.Node = &core.Node{Id: node.Id, Cluster: node.Cluster, Metadata: node.Metadata}
		if locality := node.Locality; locality != nil {
			converted.Node.Locality = &core.Locality{Region: locality.Region, Zone: locality.Zone, SubZone: locality.SubZone}
		}
	}
	// C-core registers resource kinds when their first watch is created. The
	// custom engine prefetches dependencies; retain the latest response per kind
	// until the SDK subscribes, or C-core discards it before parsing RDS/CDS.
	s.state.Lock()
	wanted := true
	if len(request.ResourceNames) > 0 {
		s.named[request.TypeUrl] = true
	} else {
		wanted = !s.named[request.TypeUrl]
	}
	active := s.watched[request.TypeUrl]
	if !wanted {
		s.watched[request.TypeUrl] = false
	}
	s.state.Unlock()
	if !wanted || active {
		return converted, nil
	}
	// Publish the watch and drain its cached response under the same writer
	// lock, so a newer push cannot overtake the older cached response.
	s.writer.Lock()
	defer s.writer.Unlock()
	s.state.Lock()
	s.watched[request.TypeUrl] = true
	pending := s.pending[request.TypeUrl]
	delete(s.pending, request.TypeUrl)
	s.state.Unlock()
	if pending != nil {
		if err := s.AggregatedDiscoveryService_StreamAggregatedResourcesServer.Send(pending); err != nil {
			return nil, err
		}
	}
	return converted, nil
}

func (s *standardStream) Send(response *discovery.DiscoveryResponse) error {
	typeURL, err := standardURL(response.TypeUrl)
	if err != nil {
		return status.Error(codes.Unimplemented, err.Error())
	}
	converted := &standard.DiscoveryResponse{VersionInfo: response.VersionInfo, TypeUrl: typeURL, Nonce: response.Nonce, Canary: response.Canary}
	if response.ControlPlane != nil {
		converted.ControlPlane = &standardcore.ControlPlane{Identifier: response.ControlPlane.Identifier}
	}
	for _, resource := range response.Resources {
		translated, err := standardAny(resource)
		if err != nil {
			return status.Errorf(codes.Unimplemented, "SDK resource conversion: %v", err)
		}
		converted.Resources = append(converted.Resources, translated)
	}
	s.writer.Lock()
	defer s.writer.Unlock()
	s.state.Lock()
	if !s.watched[typeURL] {
		s.pending[typeURL] = converted
		s.state.Unlock()
		return nil
	}
	s.state.Unlock()
	return s.AggregatedDiscoveryService_StreamAggregatedResourcesServer.Send(converted)
}

func standardAny(source *anypb.Any) (*anypb.Any, error) {
	if source == nil {
		return nil, fmt.Errorf("missing SDK resource")
	}
	typeURL, err := standardURL(source.TypeUrl)
	if err != nil {
		return nil, err
	}
	sourceType, err := protoregistry.GlobalTypes.FindMessageByURL(source.TypeUrl)
	if err != nil {
		return nil, err
	}
	destinationType, err := protoregistry.GlobalTypes.FindMessageByURL(typeURL)
	if err != nil {
		return nil, err
	}
	decoded, translated := sourceType.New(), destinationType.New()
	if err := proto.Unmarshal(source.Value, decoded.Interface()); err != nil {
		return nil, err
	}
	if err := translateMessage(decoded, translated); err != nil {
		return nil, err
	}
	return anypb.New(translated.Interface())
}

// Convert by field and enum names: the custom contract does not share all Envoy
// field numbers. Reject configured behavior that a standard SDK cannot represent.
func translateMessage(source, destination protoreflect.Message) error {
	if len(source.GetUnknown()) != 0 {
		return fmt.Errorf("unknown fields in %s", source.Descriptor().FullName())
	}
	if source.Descriptor().FullName() == "google.protobuf.Any" {
		converted, err := standardAny(source.Interface().(*anypb.Any))
		if err != nil {
			return err
		}
		proto.Merge(destination.Interface(), converted)
		return nil
	}
	var failure error
	source.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		other := destination.Descriptor().Fields().ByName(field.Name())
		if other == nil || field.IsList() != other.IsList() || field.IsMap() != other.IsMap() {
			failure = fmt.Errorf("unsupported SDK field %s", field.FullName())
			return false
		}
		switch {
		case field.IsMap():
			if field.MapKey().Kind() != other.MapKey().Kind() {
				failure = fmt.Errorf("incompatible SDK map %s", field.FullName())
				return false
			}
			target := destination.Mutable(other).Map()
			value.Map().Range(func(key protoreflect.MapKey, entry protoreflect.Value) bool {
				converted, err := translateValue(entry, field.MapValue(), other.MapValue(), target.NewValue)
				if err != nil {
					failure = err
					return false
				}
				target.Set(key, converted)
				return true
			})
		case field.IsList():
			target := destination.Mutable(other).List()
			for i := 0; i < value.List().Len(); i++ {
				converted, err := translateValue(value.List().Get(i), field, other, target.NewElement)
				if err != nil {
					failure = err
					break
				}
				target.Append(converted)
			}
		default:
			converted, err := translateValue(value, field, other, func() protoreflect.Value { return destination.NewField(other) })
			if err != nil {
				failure = err
			} else {
				destination.Set(other, converted)
			}
		}
		return failure == nil
	})
	if failure != nil {
		return failure
	}
	if config, ok := destination.Interface().(*standardcore.ConfigSource); ok {
		config.ResourceApiVersion = standardcore.ApiVersion_V3
	}
	if socket, ok := destination.Interface().(*standardcore.TransportSocket); ok && socket.Name == "transport_sockets.tls" {
		socket.Name = "envoy.transport_sockets.tls"
	}
	if context, ok := destination.Interface().(*standardtls.CommonTlsContext); ok {
		// Publish the current A29 provider fields. Java no longer reads the
		// deprecated fields carried by the custom contract.
		if provider := context.GetTlsCertificateCertificateProviderInstance(); provider != nil {
			context.TlsCertificateProviderInstance = &standardtls.CertificateProviderPluginInstance{
				InstanceName: provider.InstanceName, CertificateName: provider.CertificateName,
			}
			context.TlsCertificateCertificateProviderInstance = nil
		}
		if combined := context.GetCombinedValidationContext(); combined != nil {
			if provider := combined.GetValidationContextCertificateProviderInstance(); provider != nil {
				if combined.DefaultValidationContext == nil {
					combined.DefaultValidationContext = &standardtls.CertificateValidationContext{}
				}
				combined.DefaultValidationContext.CaCertificateProviderInstance = &standardtls.CertificateProviderPluginInstance{
					InstanceName: provider.InstanceName, CertificateName: provider.CertificateName,
				}
				combined.ValidationContextCertificateProviderInstance = nil
			}
		}
	}
	return nil
}

func translateValue(value protoreflect.Value, source, destination protoreflect.FieldDescriptor, create func() protoreflect.Value) (protoreflect.Value, error) {
	// The custom contract represents exact SAN matches as strings; standard
	// xDS represents the same constraint with StringMatcher messages.
	if source.FullName() == "extensions.transport_sockets.tls.v1.CertificateValidationContext.match_subject_alt_names" &&
		source.Kind() == protoreflect.StringKind && destination.Kind() == protoreflect.MessageKind &&
		destination.Message().FullName() == "envoy.type.matcher.v3.StringMatcher" {
		converted := create()
		matcher := converted.Message().Interface().(*standardmatcher.StringMatcher)
		matcher.MatchPattern = &standardmatcher.StringMatcher_Exact{Exact: value.String()}
		return converted, nil
	}
	if source.Kind() != destination.Kind() {
		return protoreflect.Value{}, fmt.Errorf("incompatible SDK field %s", source.FullName())
	}
	switch source.Kind() {
	case protoreflect.MessageKind:
		converted := create()
		if err := translateMessage(value.Message(), converted.Message()); err != nil {
			return protoreflect.Value{}, err
		}
		return converted, nil
	case protoreflect.EnumKind:
		enum := source.Enum().Values().ByNumber(value.Enum())
		if enum == nil {
			return protoreflect.Value{}, fmt.Errorf("unknown SDK enum in %s", source.FullName())
		}
		translated := destination.Enum().Values().ByName(enum.Name())
		if translated == nil {
			return protoreflect.Value{}, fmt.Errorf("unsupported SDK enum %s", enum.FullName())
		}
		return protoreflect.ValueOfEnum(translated.Number()), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes(append([]byte(nil), value.Bytes()...)), nil
	default:
		return value, nil
	}
}
