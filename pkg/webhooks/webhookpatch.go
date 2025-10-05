package webhooks

import (
	"bytes"
	"errors"
	kubelib "github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/util"
	"github.com/apache/dubbo-kubernetes/sail/pkg/keycertbundle"
	"istio.io/api/label"
	v1 "k8s.io/api/admissionregistration/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"math"
	"strings"
	"time"
)

var (
	errWrongRevision     = errors.New("webhook does not belong to target revision")
	errNotFound          = errors.New("webhook not found")
	errNoWebhookWithName = errors.New("webhook configuration did not contain webhook with target name")
)

type WebhookCertPatcher struct {
	// revision to patch webhooks for
	revision    string
	webhookName string

	queue controllers.Queue

	// File path to the x509 certificate bundle used by the webhook server
	// and patched into the webhook config.
	CABundleWatcher *keycertbundle.Watcher

	webhooks kclient.Client[*v1.MutatingWebhookConfiguration]
}

func NewWebhookCertPatcher(client kubelib.Client, webhookName, revision string, caBundleWatcher *keycertbundle.Watcher) (*WebhookCertPatcher, error) {
	p := &WebhookCertPatcher{
		revision:        revision,
		webhookName:     webhookName,
		CABundleWatcher: caBundleWatcher,
	}
	p.queue = newWebhookPatcherQueue(p.webhookPatchTask)

	p.webhooks = kclient.New[*v1.MutatingWebhookConfiguration](client)
	p.webhooks.AddEventHandler(controllers.ObjectHandler(p.queue.AddObject))

	return p, nil
}

func newWebhookPatcherQueue(reconciler controllers.ReconcilerFn) controllers.Queue {
	return controllers.NewQueue("webhook patcher",
		controllers.WithReconciler(reconciler),
		controllers.WithRateLimiter(workqueue.NewTypedItemFastSlowRateLimiter[any](100*time.Millisecond, 1*time.Minute, 5)),
		controllers.WithMaxAttempts(math.MaxInt))
}

func (w *WebhookCertPatcher) webhookPatchTask(o types.NamespacedName) error {
	err := w.patchMutatingWebhookConfig(o.Name)

	// do not want to retry the task if these errors occur, they indicate that
	// we should no longer be patching the given webhook
	if kerrors.IsNotFound(err) || errors.Is(err, errWrongRevision) || errors.Is(err, errNoWebhookWithName) || errors.Is(err, errNotFound) {
		return nil
	}

	if err != nil {
		klog.Errorf("patching webhook %s failed: %v", o.Name, err)
	}

	return err
}

func (w *WebhookCertPatcher) patchMutatingWebhookConfig(webhookConfigName string) error {
	config := w.webhooks.Get(webhookConfigName, "")
	if config == nil {
		return errNotFound
	}
	// prevents a race condition between multiple istiods when the revision is changed or modified
	v, ok := config.Labels[label.IoIstioRev.Name]
	if !ok {
		return nil
	}
	klog.Infof("This is webhook label: %v", v)

	if v != w.revision {
		return errWrongRevision
	}

	found := false
	updated := false
	caCertPem, err := util.LoadCABundle(w.CABundleWatcher)
	if err != nil {
		klog.Errorf("Failed to load CA bundle: %v", err)
		return err
	}
	for i, wh := range config.Webhooks {
		if strings.HasSuffix(wh.Name, w.webhookName) {
			if !bytes.Equal(caCertPem, config.Webhooks[i].ClientConfig.CABundle) {
				updated = true
			}
			config.Webhooks[i].ClientConfig.CABundle = caCertPem
			found = true
		}
	}
	if !found {
		return errNoWebhookWithName
	}

	if updated {
		_, err := w.webhooks.Update(config)
		if err != nil {
		}
	}

	return err
}

func (w *WebhookCertPatcher) startCaBundleWatcher(stop <-chan struct{}) {
	id, watchCh := w.CABundleWatcher.AddWatcher()
	defer w.CABundleWatcher.RemoveWatcher(id)
	for {
		select {
		case <-watchCh:
			for _, whc := range w.webhooks.List("", klabels.Everything()) {
				w.queue.AddObject(whc)
			}
		case <-stop:
			return
		}
	}
}

func (w *WebhookCertPatcher) Run(stopChan <-chan struct{}) {
	go w.startCaBundleWatcher(stopChan)
	w.webhooks.Start(stopChan)
	kubelib.WaitForCacheSync("webhook patcher", stopChan, w.webhooks.HasSynced)
	w.queue.Run(stopChan)
}

func (w *WebhookCertPatcher) HasSynced() bool {
	return w.queue.HasSynced()
}
