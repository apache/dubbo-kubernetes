package gvr

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	CustomResourceDefinition = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
)
