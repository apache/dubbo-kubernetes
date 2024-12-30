package install

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/uninstall"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/dmultierr"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/progress"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/pkg/util/slices"
	"github.com/hashicorp/go-multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

type Installer struct {
	DryRun       bool
	SkipWait     bool
	Kube         kube.CLIClient
	Values       values.Map
	ProgressInfo *progress.Info
	WaitTimeout  time.Duration
	Logger       clog.Logger
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

	if err := i.prune(manifests); err != nil {
		return fmt.Errorf("pruning: %v", err)
	}
	i.ProgressInfo.SetState(progress.StateComplete)
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
		obj, err := i.applyLabelsAndAnnotations(obj, componentNames)
		if err != nil {
			return err
		}
		if err := i.serverSideApply(obj); err != nil {
			pi.ReportError(err.Error())
			return err
		}
		pi.ReportProgress()
	}
	if err := WaitForResources(manifests, i.Kube, i.WaitTimeout, i.DryRun, pi); err != nil {
		werr := fmt.Errorf("failed to wait for resource: %v", err)
		pi.ReportError(werr.Error())
		return werr
	}
	pi.ReportFinished()
	return nil
}

func (i Installer) serverSideApply(obj manifest.Manifest) error {
	var dryRun []string
	const fieldManager = "dubbo-operator"
	dc, err := i.Kube.DynamicClientFor(obj.GroupVersionKind(), obj.Unstructured, "")
	if err != nil {
		return err
	}
	objStr := fmt.Sprintf("%s/%s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())

	if i.DryRun {
		return nil
	}
	if _, err := dc.Patch(context.TODO(), obj.GetName(), types.ApplyPatchType, []byte(obj.Content), metav1.PatchOptions{
		DryRun:       dryRun,
		FieldManager: fieldManager,
	}); err != nil {
		return fmt.Errorf("failed to update resource with server-side apply for obj %v: %v", objStr, err)
	}
	return nil
}

func (i Installer) applyLabelsAndAnnotations(obj manifest.Manifest, cname string) (manifest.Manifest, error) {
	for k, v := range getOwnerLabels(i.Values, cname) {
		err := util.SetLabel(obj, k, v)
		if err != nil {
			return manifest.Manifest{}, err
		}
	}
	return manifest.FromObject(obj.Unstructured)
}

func (i Installer) prune(manifests []manifest.ManifestSet) error {
	if i.DryRun {
		return nil
	}

	i.ProgressInfo.SetState(progress.StatePruning)

	excluded := map[component.Name]sets.String{}
	for _, c := range component.AllComponents {
		excluded[c.UserFacingName] = sets.New[string]()
	}

	for _, mfs := range manifests {
		for _, m := range mfs.Manifests {
			excluded[mfs.Components].Insert(m.Hash())
		}
	}

	coreLabels := getOwnerLabels(i.Values, "")
	selector := klabels.Set(coreLabels).AsSelectorPreValidated()
	compReq, err := klabels.NewRequirement(manifest.DubboComponentLabel, selection.Exists, nil)
	if err != nil {
		return err
	}
	selector = selector.Add(*compReq)

	var errs util.Errors
	resources := uninstall.PrunedResourcesSchemas()
	for _, gvk := range resources {
		dc, err := i.Kube.DynamicClientFor(gvk, nil, "")
		if err != nil {
			return err
		}
		objs, err := dc.List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
		if objs == nil {
			continue
		}
		for comp, excluded := range excluded {
			compLabels := klabels.SelectorFromSet(getOwnerLabels(i.Values, string(comp)))
			for _, obj := range objs.Items {
				if excluded.Contains(manifest.ObjectHash(&obj)) {
					continue
				}
				if obj.GetLabels()[manifest.OwningResourceNotPruned] == "true" {
					continue
				}
				if !compLabels.Matches(klabels.Set(obj.GetLabels())) {
					continue
				}
				if err := uninstall.DeleteResource(i.Kube, i.DryRun, i.Logger, &obj); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errs.ToErrors()
}

var componentDependencies = map[component.Name][]component.Name{
	component.BaseComponentName: {},
	component.AdminComponentName: {
		component.BaseComponentName,
	},
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

func getOwnerLabels(dop values.Map, c string) map[string]string {
	labels := make(map[string]string)

	if n := dop.GetPathString("metadata.name"); n != "" {
		labels[manifest.OwningResourceName] = n
	}
	if n := dop.GetPathString("metadata.namespace"); n != "" {
		labels[manifest.OwningResourceNamespace] = n
	}
	if c != "" {
		labels[manifest.DubboComponentLabel] = c
	}
	return labels
}
