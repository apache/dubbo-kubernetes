package kube

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
)

func dubboScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(kubescheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

}
