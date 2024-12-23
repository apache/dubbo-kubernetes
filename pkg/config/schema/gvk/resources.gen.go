package gvk

import "github.com/apache/dubbo-kubernetes/operator/pkg/config"

var (
	CustomResourceDefinition = config.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
)
