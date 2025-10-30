package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"istio.io/api/annotation"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
)

// ConstructProxyConfig returns proxyConfig
func ConstructProxyConfig(meshConfigFile, proxyConfigEnv string) (*meshconfig.ProxyConfig, error) {
	annotations, err := bootstrap.ReadPodAnnotations("")
	if err != nil {
		if os.IsNotExist(err) {
			klog.V(2).Infof("failed to read pod annotations: %v", err)
		} else {
			klog.V(2).Infof("failed to read pod annotations: %v", err)
		}
	}
	var fileMeshContents string
	if fileExists(meshConfigFile) {
		contents, err := os.ReadFile(meshConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read mesh config file %v: %v", meshConfigFile, err)
		}
		fileMeshContents = string(contents)
	}
	meshConfig, err := getMeshConfig(fileMeshContents, annotations["proxy.dubbo.io/config"], proxyConfigEnv)
	if err != nil {
		return nil, err
	}
	proxyConfig := mesh.DefaultProxyConfig()
	if meshConfig.DefaultConfig != nil {
		proxyConfig = meshConfig.DefaultConfig
	}
	// TODO ResolveAddr
	// TODO ValidateMeshConfigProxyConfig
	return applyAnnotations(proxyConfig, annotations), nil
}

func getMeshConfig(fileOverride, annotationOverride, proxyConfigEnv string) (*meshconfig.MeshConfig, error) {
	mc := mesh.DefaultMeshConfig()
	if fileOverride != "" {
		klog.Infof("Apply mesh config from file %v", fileOverride)
		fileMesh, err := mesh.ApplyMeshConfig(fileOverride, mc)
		if err != nil || fileMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from file [%v]: %v", fileOverride, err)
		}
		mc = fileMesh
	}

	// Original order: env first, then annotation
	if proxyConfigEnv != "" {
		klog.Infof("Apply proxy config from env %v", proxyConfigEnv)
		envMesh, err := mesh.ApplyProxyConfig(proxyConfigEnv, mc)
		if err != nil || envMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from environment [%v]: %v", proxyConfigEnv, err)
		}
		mc = envMesh
	}

	if annotationOverride != "" {
		klog.Infof("Apply proxy config from annotation %v", annotationOverride)
		annotationMesh, err := mesh.ApplyProxyConfig(annotationOverride, mc)
		if err != nil || annotationMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from annotation [%v]: %v", annotationOverride, err)
		}
		mc = annotationMesh
	}

	return mc, nil
}

// Apply any overrides to proxy config from annotations
func applyAnnotations(config *meshconfig.ProxyConfig, annos map[string]string) *meshconfig.ProxyConfig {
	if v, f := annos[annotation.SidecarDiscoveryAddress.Name]; f {
		config.DiscoveryAddress = v
	}
	if v, f := annos[annotation.SidecarStatusPort.Name]; f {
		p, err := strconv.Atoi(v)
		if err != nil {
			klog.Errorf("Invalid annotation %v=%v: %v", annotation.SidecarStatusPort.Name, v, err)
		}
		config.StatusPort = int32(p)
	}
	return config
}

func GetSailSan(discoveryAddress string) string {
	discHost := strings.Split(discoveryAddress, ":")[0]
	// For local debugging - the discoveryAddress is set to localhost, but the cert issued for normal SA.
	if discHost == "localhost" {
		discHost = "dubbod.istio-system.svc"
	}
	return discHost
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

var CPULimit = env.Register(
	"DUBBO_CPU_LIMIT",
	0,
	"CPU limit for the current process. Expressed as an integer value, rounded up.",
).Get()
