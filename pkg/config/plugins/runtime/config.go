package runtime

import (
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/runtime/k8s"
)

func DefaultRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		Kubernetes: k8s.DefaultKubernetesRuntimeConfig(),
	}
}

// RuntimeConfig defines Environment-specific configuration
type RuntimeConfig struct {
	// Kubernetes-specific configuration
	Kubernetes *k8s.KubernetesRuntimeConfig `json:"kubernetes"`
}

func (c *RuntimeConfig) Sanitize() {
	c.Kubernetes.Sanitize()
}

func (c *RuntimeConfig) PostProcess() error {
	return multierr.Combine(
		c.Kubernetes.PostProcess(),
	)
}

func (c *RuntimeConfig) Validate(env core.EnvironmentType) error {
	switch env {
	case core.KubernetesEnvironment:
		if err := c.Kubernetes.Validate(); err != nil {
			return errors.Wrap(err, "Kubernetes validation failed")
		}
	case core.UniversalEnvironment:
	default:
		return errors.Errorf("unknown environment type %q", env)
	}
	return nil
}
