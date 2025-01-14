package collections

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/schema"
	"github.com/apache/dubbo-kubernetes/pkg/kube/collection"
	k8sioapiextensionsapiserverpkgapisapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

var (
	CustomResourceDefinition = schema.Builder{
		Identifier:    "CustomResourceDefinition",
		Group:         "apiextensions.k8s.io",
		Kind:          "CustomResourceDefinition",
		Plural:        "customresourcedefinitions",
		Version:       "v1",
		Proto:         "k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.CustomResourceDefinition",
		ReflectType:   reflect.TypeOf(&k8sioapiextensionsapiserverpkgapisapiextensionsv1.CustomResourceDefinition{}).Elem(),
		ProtoPackage:  "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1",
		ClusterScoped: true,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()

	All = collection.NewSchemasBuilder().
		MustAdd(CustomResourceDefinition).
		Build()
)
