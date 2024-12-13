package apis

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +k8s:deepcopy-gen=package,register
// +groupName=install.dubbo.io
var DubboOperatorGVK = schema.GroupVersionKind{
	Version: "v1alpha1",
	Group:   "install.dubbo.io",
	Kind:    "DubboOperator",
}
