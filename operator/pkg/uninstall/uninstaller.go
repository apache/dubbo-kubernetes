package uninstall

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/pointer"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
)

var (
	ClusterResources    = []schema.GroupVersionKind{}
	ClusterCPResources  = []schema.GroupVersionKind{}
	AllClusterResources = append(ClusterResources,
		gvk.CustomResourceDefinition.Kubernetes())
)

func GetPrunedResources(kc kube.CLIClient, dopName, dopNamespace string, includeClusterResources bool) ([]*unstructured.UnstructuredList, error) {
	var usList []*unstructured.UnstructuredList
	labels := make(map[string]string)
	if dopName != "" {
		labels[manifest.OwningResourceName] = dopName
	}
	if dopNamespace != "" {
		labels[manifest.OwningResourceNamespace] = dopNamespace
	}
	resources := NamespacedResources()
	gvkList := append(resources, ClusterCPResources...)
	if includeClusterResources {
		gvkList = append(resources, AllClusterResources...)
	}
	for _, g := range gvkList {
		var result *unstructured.UnstructuredList
		compReq, err := klabels.NewRequirement(manifest.DubboComponentLabel, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		c, err := kc.DynamicClientFor(g, nil, "")
		if err != nil {
			return nil, err
		}
		if includeClusterResources {
			s := klabels.NewSelector()
			result, err = c.List(context.Background(), metav1.ListOptions{LabelSelector: s.Add(*compReq).String()})
		}
		if result == nil || len(result.Items) == 0 {
			continue
		}
		usList = append(usList, result)

	}
	return usList, nil
}

func NamespacedResources() []schema.GroupVersionKind {
	res := []schema.GroupVersionKind{}
	return res
}

func PrunedResourcesSchemas() []schema.GroupVersionKind {
	return append(NamespacedResources(), ClusterResources...)
}

func DeleteObjectsList(c kube.CLIClient, dryRun bool, log clog.Logger, objectsList []*unstructured.UnstructuredList) error {
	var errs util.Errors
	for _, ol := range objectsList {
		for _, o := range ol.Items {
			if err := DeleteResource(c, dryRun, log, &o); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs.ToErrors()
}

func DeleteResource(kc kube.CLIClient, dryRun bool, log clog.Logger, obj *unstructured.Unstructured) error {
	name := fmt.Sprintf("%v/%s.%s", obj.GroupVersionKind(), obj.GetName(), obj.GetNamespace())
	if dryRun {
		log.LogAndPrintf("Not pruning object %s because of dry run.", name)
		return nil
	}

	c, err := kc.DynamicClientFor(obj.GroupVersionKind(), obj, "")
	if err != nil {
		return err
	}

	if err := c.Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{PropagationPolicy: pointer.Of(metav1.DeletePropagationForeground)}); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		log.LogAndPrintf("object: %s is not being deleted because it no longer exists", name)
		return nil
	}

	log.LogAndPrintf(" ✔︎ Removed %s.", name)
	return nil
}
