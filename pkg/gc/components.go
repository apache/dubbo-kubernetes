package gc

import (
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"time"
)

func Setup(rt runtime.Runtime) error {
	if err := setupCollector(rt); err != nil {
		return err
	}
	return nil
}

func setupCollector(rt runtime.Runtime) error {
	if rt.Config().Environment != config_core.UniversalEnvironment || rt.Config().Mode == config_core.Global {
		// Dataplane GC is run only on Universal because on Kubernetes Dataplanes are bounded by ownership to Pods.
		// Therefore, on K8S offline dataplanes are cleaned up quickly enough to not run this.
		return nil
	}
	collector, err := NewCollector(
		rt.ResourceManager(),
		func() *time.Ticker { return time.NewTicker(1 * time.Minute) },
		rt.Config().Runtime.Universal.DataplaneCleanupAge.Duration,
	)
	if err != nil {
		return err
	}
	return rt.Add(collector)
}
