package installer

import (
	"github.com/apache/dubbo-kubernetes/operator/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/dmultierr"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"sync"
)

type Installer struct {
	DryRun   bool
	SkipWait bool
	Kube     kube.CLIClient
	Values   values.Map
}

func (i Installer) install(manifest []manifest.ManifestSet) error {
	var _ sync.Mutex
	var _ sync.WaitGroup
	errors := dmultierr.NewDMultiErr()
	if err := errors.ErrorOrNil(); err != nil {
		return err
	}
	return nil
}

func (i Installer) InstallManifests(manifests []manifest.ManifestSet) error {
	return nil
}
