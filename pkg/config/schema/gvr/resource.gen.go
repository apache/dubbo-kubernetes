package gvr

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	CustomResourceDefinition = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	Deployment               = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	StatefulSet              = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
)
