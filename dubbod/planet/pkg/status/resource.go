package status

import (
	"context"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
)

type Resource struct {
	schema.GroupVersionResource
	Namespace  string
	Name       string
	Generation string
}

func (r Resource) String() string {
	return strings.Join([]string{r.Group, r.Version, r.GroupVersionResource.Resource, r.Namespace, r.Name, r.Generation}, "/")
}

func (r *Resource) ToModelKey() string {
	gk, ok := gvk.FromGVR(r.GroupVersionResource)
	if !ok {
		return ""
	}
	return config.Key(
		gk.Group, gk.Version, gk.Kind,
		r.Name, r.Namespace)
}

func ResourceFromModelConfig(c config.Config) Resource {
	gvr, ok := gvk.ToGVR(c.GroupVersionKind)
	if !ok {
		return Resource{}
	}
	return Resource{
		GroupVersionResource: gvr,
		Namespace:            c.Namespace,
		Name:                 c.Name,
		Generation:           strconv.FormatInt(c.Generation, 10),
	}
}

func GetStatusManipulator(in any) (out Manipulator) {
	return &NopStatusManipulator{in}
}

func NewIstioContext(stop <-chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stop
		cancel()
	}()
	return ctx
}
