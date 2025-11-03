package clustertrustbundle

import (
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/sail/pkg/keycertbundle"
)

const (
	dubboClusterTrustBundleName       = "dubbo.io:dubbo-ca:root-cert"
	dubboClusterTrustBundleSignerName = "dubbo.io/dubbod-ca"
	maxRetries                        = 5
)

// Controller manages the lifecycle of the Istio root certificate in ClusterTrustBundle
// It watches Istio's KeyCert bundle as well as ClusterTrustBundles in the cluster, and it updates ClusterTrustBundles whenever
// the certificate is rotated. Later on, we might want to add reading from other ClusterTrustBundles as well
type Controller struct {
	caBundleWatcher     *keycertbundle.Watcher
	queue               controllers.Queue
	clustertrustbundles kclient.Client[*certificatesv1beta1.ClusterTrustBundle]
}

// NewController creates a new ClusterTrustBundleController
func NewController(kubeClient kube.Client, caBundleWatcher *keycertbundle.Watcher) *Controller {
	c := &Controller{
		caBundleWatcher: caBundleWatcher,
	}

	c.queue = controllers.NewQueue("clustertrustbundle controller",
		controllers.WithReconciler(c.reconcileClusterTrustBundle),
		controllers.WithMaxAttempts(maxRetries))

	c.clustertrustbundles = kclient.NewFiltered[*certificatesv1beta1.ClusterTrustBundle](kubeClient, kclient.Filter{
		FieldSelector: "metadata.name=" + dubboClusterTrustBundleName,
		ObjectFilter:  kubeClient.ObjectFilter(),
	})

	c.clustertrustbundles.AddEventHandler(controllers.FilteredObjectSpecHandler(c.queue.AddObject, func(o controllers.Object) bool { return true }))

	return c
}

// startCaBundleWatcher listens for updates to the CA bundle and queues a reconciliation of the ClusterTrustBundle
func (c *Controller) startCaBundleWatcher(stop <-chan struct{}) {
	id, watchCh := c.caBundleWatcher.AddWatcher()
	defer c.caBundleWatcher.RemoveWatcher(id)
	for {
		select {
		case <-watchCh:
			c.queue.AddObject(&certificatesv1beta1.ClusterTrustBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name: dubboClusterTrustBundleName,
				},
			})
		case <-stop:
			return
		}
	}
}

func (c *Controller) reconcileClusterTrustBundle(o types.NamespacedName) error {
	if o.Name == dubboClusterTrustBundleName {
		return c.updateClusterTrustBundle(c.caBundleWatcher.GetCABundle())
	}
	return nil
}

// Run starts the controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	if !kube.WaitForCacheSync("clustertrustbundle controller", stopCh, c.clustertrustbundles.HasSynced) {
		return
	}
	go c.startCaBundleWatcher(stopCh)

	// queue an initial event
	c.queue.AddObject(&certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: dubboClusterTrustBundleName,
		},
	})

	c.queue.Run(stopCh)
	controllers.ShutdownAll(c.clustertrustbundles)
}

// updateClusterTrustBundle updates the root certificate in the ClusterTrustBundle
func (c *Controller) updateClusterTrustBundle(rootCert []byte) error {
	bundle := &certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: dubboClusterTrustBundleName,
		},
		Spec: certificatesv1beta1.ClusterTrustBundleSpec{
			SignerName:  dubboClusterTrustBundleSignerName,
			TrustBundle: string(rootCert),
		},
	}

	existing := c.clustertrustbundles.Get(dubboClusterTrustBundleName, "")
	if existing != nil {
		if existing.Spec.TrustBundle == string(rootCert) {
			// trustbundle is up to date. nothing to do
			return nil
		}
		// Update existing bundle
		existing.Spec.TrustBundle = string(rootCert)
		_, err := c.clustertrustbundles.Update(existing)
		return err
	}

	// Create new bundle
	_, err := c.clustertrustbundles.Create(bundle)
	return err
}
