package controller

import (
	"bytes"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/util"
	"github.com/apache/dubbo-kubernetes/sail/pkg/keycertbundle"
	"istio.io/api/label"
	kubeApiAdmission "k8s.io/api/admissionregistration/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"math"
	"time"
)

type Options struct {
	// Istio system namespace where istiod resides.
	WatchedNamespace string

	// File path to the x509 certificate bundle used by the webhook server
	// and patched into the webhook config.
	CABundleWatcher *keycertbundle.Watcher

	// Revision for control plane performing patching on the validating webhook.
	Revision string

	// Name of the service running the webhook server.
	ServiceName string
}

type Controller struct {
	o      Options
	client kube.Client

	queue                         controllers.Queue
	dryRunOfInvalidConfigRejected bool
	webhooks                      kclient.Client[*kubeApiAdmission.ValidatingWebhookConfiguration]
}

func NewValidatingWebhookController(client kube.Client, revision, ns string, caBundleWatcher *keycertbundle.Watcher) *Controller {
	o := Options{
		Revision:         revision,
		WatchedNamespace: ns,
		CABundleWatcher:  caBundleWatcher,
		ServiceName:      "dubbod",
	}
	return newController(o, client)
}

func caBundleUpdateRequired(current *kubeApiAdmission.ValidatingWebhookConfiguration, caBundle []byte) bool {
	for _, wh := range current.Webhooks {
		if !bytes.Equal(wh.ClientConfig.CABundle, caBundle) {
			return true
		}
	}
	return false
}

func failurePolicyIsIgnore(current *kubeApiAdmission.ValidatingWebhookConfiguration) bool {
	for _, wh := range current.Webhooks {
		if wh.FailurePolicy != nil && *wh.FailurePolicy != kubeApiAdmission.Fail {
			return true
		}
	}
	return false
}

func (c *Controller) readyForFailClose() bool {
	if !c.dryRunOfInvalidConfigRejected {

		// Sync all webhooks; this ensures if we have multiple webhooks all of them are updated
		c.syncAll()
	}
	return true
}

func (c *Controller) updateValidatingWebhookConfiguration(current *kubeApiAdmission.ValidatingWebhookConfiguration, caBundle []byte) error {
	caChangeNeeded := caBundleUpdateRequired(current, caBundle)
	failurePolicyMaybeNeedsUpdate := failurePolicyIsIgnore(current)
	if !caChangeNeeded && !failurePolicyMaybeNeedsUpdate {
		klog.V(2).Info("up-to-date, no change required")
		return nil
	}
	updateFailurePolicy := true
	// Only check readyForFailClose if we need to switch, to avoid redundant calls
	if failurePolicyMaybeNeedsUpdate && !c.readyForFailClose() {
		klog.V(2).Info("failurePolicy is Ignore, but webhook is not ready; not setting to Fail")
		updateFailurePolicy = false
	}
	updated := current.DeepCopy()
	for i := range updated.Webhooks {
		updated.Webhooks[i].ClientConfig.CABundle = caBundle
		if updateFailurePolicy {
			updated.Webhooks[i].FailurePolicy = ptr.Of(kubeApiAdmission.Fail)
		}
	}

	_, err := c.webhooks.Update(updated)
	if err != nil {
		klog.Errorf("failed to updated: %v", err)
		return fmt.Errorf("fail to update webhook: %v", err)
	}

	if !updateFailurePolicy {
		return fmt.Errorf("webhook is not ready, retry")
	}
	return nil
}

func (c *Controller) Reconcile(key types.NamespacedName) error {
	name := key.Name
	whc := c.webhooks.Get(name, "")
	// Stop early if webhook is not present, rather than attempting (and failing) to reconcile permanently
	// If the webhook is later added a new reconciliation request will trigger it to update
	if whc == nil {
		klog.Info("Skip patching webhook, not found")
		return nil
	}

	klog.V(2).Info("Reconcile(enter)")
	defer func() { klog.V(2).Info("Reconcile(exit)") }()

	caBundle, err := util.LoadCABundle(c.o.CABundleWatcher)
	if err != nil {
		klog.Errorf("Failed to load CA bundle: %v", err)
		return nil
	}
	return c.updateValidatingWebhookConfiguration(whc, caBundle)
}

func newController(o Options, client kube.Client) *Controller {
	c := &Controller{
		o:      o,
		client: client,
	}

	c.queue = controllers.NewQueue("validation",
		controllers.WithReconciler(c.Reconcile),
		// Webhook patching has to be retried forever. But the retries would be rate limited.
		controllers.WithMaxAttempts(math.MaxInt),
		// Retry with backoff. Failures could be from conflicts of other instances (quick retry helps), or
		// longer lasting concerns which will eventually be retried on 1min interval.
		// Unlike the mutating webhook controller, we do not use NewItemFastSlowRateLimiter. This is because
		// the validation controller waits for its own service to be ready, so typically this takes a few seconds
		// before we are ready; using FastSlow means we tend to always take the Slow time (1min).
		controllers.WithRateLimiter(workqueue.NewTypedItemExponentialFailureRateLimiter[any](100*time.Millisecond, 1*time.Minute)))

	c.webhooks = kclient.NewFiltered[*kubeApiAdmission.ValidatingWebhookConfiguration](client, kclient.Filter{
		LabelSelector: fmt.Sprintf("%s=%s", label.IoIstioRev.Name, o.Revision),
	})
	c.webhooks.AddEventHandler(controllers.ObjectHandler(c.queue.AddObject))

	return c
}

func (c *Controller) Run(stop <-chan struct{}) {
	kube.WaitForCacheSync("validation", stop, c.webhooks.HasSynced)
	go c.startCaBundleWatcher(stop)
	c.queue.Run(stop)
}

func (c *Controller) startCaBundleWatcher(stop <-chan struct{}) {
	if c.o.CABundleWatcher == nil {
		return
	}
	id, watchCh := c.o.CABundleWatcher.AddWatcher()
	defer c.o.CABundleWatcher.RemoveWatcher(id)

	for {
		select {
		case <-watchCh:
			c.syncAll()
		case <-stop:
			return
		}
	}
}

func (c *Controller) syncAll() {
	for _, whc := range c.webhooks.List("", klabels.Everything()) {
		c.queue.AddObject(whc)
	}
}
