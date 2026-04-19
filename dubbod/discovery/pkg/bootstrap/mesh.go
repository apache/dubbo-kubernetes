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
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	"os"
	"reflect"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/kubemesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	dubbokeepalive "github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/log"
)

const (
	defaultMeshGlobalConfigMapName = "dubbo"
)

func (s *Server) initMeshGlobalConfiguration(args *DubboArgs, fileWatcher filewatcher.FileWatcher) {
	log.Infof("initializing mesh global configuration %v", args.MeshGlobalConfigFile)
	col := s.getMeshGlobalConfiguration(args, fileWatcher)
	col.AsCollection().WaitUntilSynced(s.internalStop)
	s.environment.Watcher = meshwatcher.ConfigAdapter(col)
	log.Infof("mesh global configuration: %s", compactMeshGlobalConfig(s.environment.Mesh()))
	log.Infof("flags: %s", compactDubboArgs(args))
}

func (s *Server) getMeshGlobalConfiguration(args *DubboArgs, fileWatcher filewatcher.FileWatcher) krt.Singleton[meshwatcher.MeshGlobalConfigResource] {
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	sources := s.getConfigurationSources(args, fileWatcher, args.MeshGlobalConfigFile, kubemesh.MeshGlobalConfigKey)
	if len(sources) == 0 {
		fmt.Printf("\nUsing default mesh - missing file %s and no k8s client\n", args.MeshGlobalConfigFile)
	}
	return meshwatcher.NewCollection(opts, sources...)
}

func (s *Server) getConfigurationSources(args *DubboArgs, fileWatcher filewatcher.FileWatcher, file string, cmKey string) []meshwatcher.MeshGlobalConfigSource {
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	var userMeshGlobalConfig *meshwatcher.MeshGlobalConfigSource
	if features.SharedMeshGlobalConfig != "" && s.kubeClient != nil {
		userMeshGlobalConfig = ptr.Of(kubemesh.NewConfigMapSource(s.kubeClient, args.Namespace, features.SharedMeshGlobalConfig, cmKey, opts))
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		fileSource, err := meshwatcher.NewFileSource(fileWatcher, file, opts)
		if err == nil {
			return toSources(fileSource, userMeshGlobalConfig)
		}
	}

	if s.kubeClient == nil {
		return nil
	}

	configMapName := getMeshGlobalConfigMapName("")
	primary := kubemesh.NewConfigMapSource(s.kubeClient, args.Namespace, configMapName, cmKey, opts)
	return toSources(primary, userMeshGlobalConfig)
}

func toSources(base meshwatcher.MeshGlobalConfigSource, user *meshwatcher.MeshGlobalConfigSource) []meshwatcher.MeshGlobalConfigSource {
	if user != nil {
		return []meshwatcher.MeshGlobalConfigSource{*user, base}
	}
	return []meshwatcher.MeshGlobalConfigSource{base}
}

func getMeshGlobalConfigMapName(revision string) string {
	name := defaultMeshGlobalConfigMapName
	if revision == "" || revision == "default" {
		return name
	}
	return name + "-" + revision
}

func compactMeshGlobalConfig(cfg *meshv1alpha1.MeshGlobalConfig) string {
	if cfg == nil {
		return "<nil>"
	}

	parts := []string{
		formatDurationField("connect_timeout", cfg.GetConnectTimeout()),
		formatStringField("trust_domain", cfg.GetTrustDomain()),
		formatStringField("root_namespace", cfg.GetRootNamespace()),
		formatDurationField("dns_refresh_rate", cfg.GetDnsRefreshRate()),
		formatStringSliceField("service_export_to", cfg.GetDefaultServiceExportTo()),
		formatStringSliceField("virtual_service_export_to", cfg.GetDefaultVirtualServiceExportTo()),
		formatStringSliceField("destination_rule_export_to", cfg.GetDefaultDestinationRuleExportTo()),
	}

	if defaultCfg := cfg.GetDefaultConfig(); defaultCfg != nil {
		parts = append(parts,
			formatStringField("config_path", defaultCfg.GetConfigPath()),
			formatStringField("discovery_address", defaultCfg.GetDiscoveryAddress()),
			formatEnumField("control_plane_auth_policy", defaultCfg.GetControlPlaneAuthPolicy().String()),
			formatInt32Field("status_port", defaultCfg.GetStatusPort()),
		)
	}

	return compactParts(parts...)
}

func compactDubboArgs(args *DubboArgs) string {
	if args == nil {
		return "<nil>"
	}

	parts := []string{
		formatStringField("mesh_file", args.MeshGlobalConfigFile),
		formatQuotedField("namespace", args.Namespace),
		formatQuotedField("pod_name", args.PodName),
		formatQuotedField("revision", args.Revision),
		formatStringSliceField("registries", args.RegistryOptions.Registries),
		formatQuotedField("cluster_registries_namespace", args.RegistryOptions.ClusterRegistriesNamespace),
		formatQuotedField("kubeconfig", args.RegistryOptions.KubeConfig),
		formatQuotedField("config_dir", args.RegistryOptions.FileDir),
		formatStringField("injection_dir", args.InjectionOptions.InjectionDirectory),
		formatStringField("ctrlz", ctrlzAddress(args)),
		formatCompositeField("server", serverAddressSummary(args)),
		formatCompositeField("kube", kubeOptionsSummary(args)),
		formatCompositeField("keepalive", keepaliveSummary(args)),
	}

	return compactParts(parts...)
}

func ctrlzAddress(args *DubboArgs) string {
	if args == nil || args.CtrlZOptions == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", args.CtrlZOptions.Address, args.CtrlZOptions.Port)
}

func serverAddressSummary(args *DubboArgs) string {
	if args == nil {
		return ""
	}
	return compactParts(
		formatStringField("http", args.ServerOptions.HTTPAddr),
		formatStringField("https", args.ServerOptions.HTTPSAddr),
		formatStringField("grpc", args.ServerOptions.GRPCAddr),
		formatStringField("secure_grpc", args.ServerOptions.SecureGRPCAddr),
	)
}

func kubeOptionsSummary(args *DubboArgs) string {
	if args == nil {
		return ""
	}
	return compactParts(
		formatEnumField("cluster_id", string(args.RegistryOptions.KubeOptions.ClusterID)),
		formatStringField("domain", args.RegistryOptions.KubeOptions.DomainSuffix),
		formatFloat32Field("qps", args.RegistryOptions.KubeOptions.KubernetesAPIQPS),
		formatIntField("burst", args.RegistryOptions.KubeOptions.KubernetesAPIBurst),
		formatIntField("cluster_aliases", len(args.RegistryOptions.KubeOptions.ClusterAliases)),
	)
}

func keepaliveSummary(args *DubboArgs) string {
	if args == nil || args.KeepaliveOptions == nil {
		return ""
	}
	maxAge := args.KeepaliveOptions.MaxServerConnectionAge.String()
	if args.KeepaliveOptions.MaxServerConnectionAge == dubbokeepalive.Infinity {
		maxAge = "infinity"
	}
	return compactParts(
		formatEnumField("time", args.KeepaliveOptions.Time.String()),
		formatEnumField("timeout", args.KeepaliveOptions.Timeout.String()),
		formatEnumField("max_age", maxAge),
		formatEnumField("max_age_grace", args.KeepaliveOptions.MaxServerConnectionAgeGrace.String()),
	)
}

func compactParts(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, " ")
}

func formatStringField(name, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s=%s", name, value)
}

func formatQuotedField(name, value string) string {
	return fmt.Sprintf("%s=%q", name, value)
}

func formatEnumField(name, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s=%s", name, value)
}

func formatCompositeField(name, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s={%s}", name, value)
}

func formatDurationField(name string, value interface{ String() string }) string {
	if value == nil {
		return ""
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return ""
	}
	return fmt.Sprintf("%s=%s", name, value.String())
}

func formatStringSliceField(name string, values []string) string {
	if len(values) == 0 {
		return ""
	}
	return fmt.Sprintf("%s=[%s]", name, strings.Join(values, ","))
}

func formatIntField(name string, value int) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%s=%d", name, value)
}

func formatInt32Field(name string, value int32) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%s=%d", name, value)
}

func formatFloat32Field(name string, value float32) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%s=%g", name, value)
}
