package installer

import (
	"github.com/apache/dubbo-kubernetes/operator/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/dmultierr"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/progress"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"github.com/hashicorp/go-multierror"
	"sync"
)

type Installer struct {
	DryRun       bool
	SkipWait     bool
	Kube         kube.CLIClient
	Values       values.Map
	ProgressInfo *progress.Info
}

func (i Installer) install(manifests []manifest.ManifestSet) error {
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := dmultierr.New()
	if err := errors.ErrorOrNil(); err != nil {
		return err
	}

	disabledComponents := sets.New(slices.Map(
		component.AllComponents,
		func(cc component.Component) component.Name {
			return cc.UserFacingName
		},
	)...)
	dependencyWaitCh := dependenciesChs()
	for _, mfs := range manifests {
		mfs := mfs
		c := mfs.Components
		m := mfs.Manifests
		disabledComponents.Delete(c)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s := dependencyWaitCh[c]; s != nil {
				<-s
			}
			if len(m) != 0 {
				if err := i.applyManifestSet(mfs); err != nil {
					mu.Lock()
					errors = multierror.Append(errors, err)
					mu.Unlock()
				}
			}
			for _, ch := range componentDependencies[c] {
				dependencyWaitCh[ch] <- struct{}{}
			}
		}()
	}
	for cc := range disabledComponents {
		for _, ch := range componentDependencies[cc] {
			dependencyWaitCh[ch] <- struct{}{}
		}
	}
	wg.Wait()
	if err := errors.ErrorOrNil(); err != nil {
		return err
	}
	return nil
}

func (i Installer) InstallManifests(manifests []manifest.ManifestSet) error {
	if err := i.install(manifests); err != nil {
		return err
	}
	return nil
}

func (i Installer) applyManifestSet(manifestSet manifest.ManifestSet) error {
	componentNames := string(manifestSet.Components)
	manifests := manifestSet.Manifests
	pi := i.ProgressInfo.NewComponent(componentNames)
	for _, obj := range manifests {
		if err := i.serverSideApply(obj); err != nil {
			pi.ReportError(err.Error())
			return err
		}
		pi.ReportProgress()
	}
	pi.ReportFinished()
	return nil
}

func (i Installer) serverSideApply(obj manifest.Manifest) error {
	const fieldOwner = "dubbo-operator"
	return nil
}

var componentDependencies = map[component.Name][]component.Name{
	component.BaseComponentName: {},
}

func dependenciesChs() map[component.Name]chan struct{} {
	r := make(map[component.Name]chan struct{})
	for _, parent := range componentDependencies {
		for _, child := range parent {
			r[child] = make(chan struct{}, 1)
		}
	}
	return r
}
