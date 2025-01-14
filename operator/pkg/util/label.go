package util

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func SetLabel(resource runtime.Object, label, value string) error {
	resourceAccessor, err := meta.Accessor(resource)
	if err != nil {
		return err
	}
	labels := resourceAccessor.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[label] = value
	resourceAccessor.SetLabels(labels)
	return nil
}
