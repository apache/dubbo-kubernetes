package k8s

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

func DefaultKubernetesStoreConfig() *KubernetesStoreConfig {
	return &KubernetesStoreConfig{
		SystemNamespace: "dubbo-system",
	}
}

var _ config.Config = &KubernetesStoreConfig{}

// KubernetesStoreConfig defines Kubernetes store configuration
type KubernetesStoreConfig struct {
	config.BaseConfig

	// Namespace where Control Plane is installed to.
	SystemNamespace string `json:"systemNamespace" envconfig:"dubbo_store_kubernetes_system_namespace"`
}

func (p *KubernetesStoreConfig) Validate() error {
	if len(p.SystemNamespace) < 1 {
		return errors.New("SystemNamespace should not be empty")
	}
	return nil
}
