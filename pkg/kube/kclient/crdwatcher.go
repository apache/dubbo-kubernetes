package kclient

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sync"
)

type crdWatcher struct {
	crds      Informer[*metav1.PartialObjectMetadata]
	queue     controllers.Queue
	mutex     sync.RWMutex
	callbacks map[string][]func()

	running chan struct{}
	stop    <-chan struct{}
}

func init() {
	// Unfortunate hack needed to avoid circular imports
	kube.NewCrdWatcher = newCrdWatcher
}

func newCrdWatcher(client kube.Client) kubetypes.CrdWatcher {
	c := &crdWatcher{
		running:   make(chan struct{}),
		callbacks: map[string][]func(){},
	}

	c.queue = controllers.NewQueue("crd watcher",
		controllers.WithReconciler(c.Reconcile))
	c.crds = NewMetadata(client, gvr.CustomResourceDefinition, Filter{
		ObjectFilter: kubetypes.NewStaticObjectFilter(minimumVersionFilter),
	})
	c.crds.AddEventHandler(controllers.ObjectHandler(c.queue.AddObject))
	return c
}

func minimumVersionFilter(t any) bool {
	return true
}

func (c *crdWatcher) Reconcile(key types.NamespacedName) error {
	c.mutex.Lock()
	callbacks, f := c.callbacks[key.Name]
	if !f {
		c.mutex.Unlock()
		return nil
	}
	// Delete them so we do not run again
	delete(c.callbacks, key.Name)
	c.mutex.Unlock()
	for _, cb := range callbacks {
		cb()
	}
	return nil
}

func (c *crdWatcher) known(s schema.GroupVersionResource) bool {
	// From the spec: "Its name MUST be in the format <.spec.name>.<.spec.group>."
	name := fmt.Sprintf("%s.%s", s.Resource, s.Group)
	return c.crds.Get(name, "") != nil
}

func (c *crdWatcher) KnownOrCallback(s schema.GroupVersionResource, f func(stop <-chan struct{})) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// If we are already synced, return immediately if the CRD is present.
	if c.crds.HasSynced() && c.known(s) {
		// Already known, return early
		return true
	}
	name := fmt.Sprintf("%s.%s", s.Resource, s.Group)
	c.callbacks[name] = append(c.callbacks[name], func() {
		// Call the callback
		f(c.stop)
	})
	return false
}

func (c *crdWatcher) WaitForCRD(s schema.GroupVersionResource, stop <-chan struct{}) bool {
	done := make(chan struct{})
	if c.KnownOrCallback(s, func(stop <-chan struct{}) {
		close(done)
	}) {
		// Already known
		return true
	}
	select {
	case <-stop:
		return false
	case <-done:
		return true
	}
}

func (c *crdWatcher) HasSynced() bool {
	return c.queue.HasSynced()
}

// Run starts the controller. This must be called.
func (c *crdWatcher) Run(stop <-chan struct{}) {
	c.mutex.Lock()
	if c.stop != nil {
		// Run already called. Because we call this from client.RunAndWait this isn't uncommon
		c.mutex.Unlock()
		return
	}
	c.stop = stop
	c.mutex.Unlock()
	kube.WaitForCacheSync("crd watcher", stop, c.crds.HasSynced)
	c.queue.Run(stop)
	c.crds.ShutdownHandlers()
}
