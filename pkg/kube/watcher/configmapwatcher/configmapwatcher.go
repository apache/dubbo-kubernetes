package configmapwatcher

import (
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"go.uber.org/atomic"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
)

type Controller struct {
	configmaps kclient.Client[*v1.ConfigMap]
	queue      controllers.Queue

	configMapNamespace string
	configMapName      string
	callback           func(*v1.ConfigMap)

	hasSynced atomic.Bool
}

func NewController(client kube.Client, namespace, name string, callback func(*v1.ConfigMap)) *Controller {
	c := &Controller{
		configMapNamespace: namespace,
		configMapName:      name,
		callback:           callback,
	}

	c.configmaps = kclient.NewFiltered[*v1.ConfigMap](client, kclient.Filter{
		Namespace:     namespace,
		FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, name).String(),
	})

	c.queue = controllers.NewQueue("configmap "+name, controllers.WithReconciler(c.processItem))
	c.configmaps.AddEventHandler(controllers.FilteredObjectSpecHandler(c.queue.AddObject, func(o controllers.Object) bool {
		// Filter out other configmaps
		return o.GetName() == name && o.GetNamespace() == namespace
	}))

	return c
}

func (c *Controller) processItem(name types.NamespacedName) error {
	cm := c.configmaps.Get(name.Name, name.Namespace)
	c.callback(cm)

	c.hasSynced.Store(true)
	return nil
}

func (c *Controller) Run(stop <-chan struct{}) {
	// Start informer immediately instead of with the rest. This is because we use configmapwatcher for
	// single types (so its never shared), and for use cases where we need the results immediately
	// during startup.
	c.configmaps.Start(stop)
	if !kube.WaitForCacheSync("configmap "+c.configMapName, stop, c.configmaps.HasSynced) {
		c.queue.ShutDownEarly()
		return
	}
	c.queue.Run(stop)
}
