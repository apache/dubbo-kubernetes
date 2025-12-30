//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"bytes"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	"math"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/keycertbundle"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/util"
	kubeApiAdmission "k8s.io/api/admissionregistration/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("webhookvalidation", "webhook validation debugging")

type Options struct {
	WatchedNamespace string
	CABundleWatcher  *keycertbundle.Watcher
	Revision         string
	ServiceName      string
}

type Controller struct {
	o                             Options
	client                        kube.Client
	queue                         controllers.Queue
	dryRunOfInvalidConfigRejected bool
	webhooks                      kclient.Client[*kubeApiAdmission.ValidatingWebhookConfiguration]
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
		LabelSelector: fmt.Sprintf("%s=%s", "dubbo.apache.org/rev", o.Revision),
	})
	c.webhooks.AddEventHandler(controllers.ObjectHandler(c.queue.AddObject))

	return c
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

func (c *Controller) Reconcile(key types.NamespacedName) error {
	name := key.Name
	whc := c.webhooks.Get(name, "")
	// Stop early if webhook is not present, rather than attempting (and failing) to reconcile permanently
	// If the webhook is later added a new reconciliation request will trigger it to update
	if whc == nil {
		log.Info("Skip patching webhook, not found")
		return nil
	}

	log.Debug("Reconcile(enter)")
	defer func() { log.Debug("Reconcile(exit)") }()

	caBundle, err := util.LoadCABundle(c.o.CABundleWatcher)
	if err != nil {
		log.Errorf("Failed to load CA bundle: %v", err)
		return nil
	}
	return c.updateValidatingWebhookConfiguration(whc, caBundle)
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
		log.Debug("up-to-date, no change required")
		return nil
	}
	updateFailurePolicy := true
	// Only check readyForFailClose if we need to switch, to avoid redundant calls
	if failurePolicyMaybeNeedsUpdate && !c.readyForFailClose() {
		log.Debug("failurePolicy is Ignore, but webhook is not ready; not setting to Fail")
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
		log.Errorf("failed to updated: %v", err)
		return fmt.Errorf("fail to update webhook: %v", err)
	}

	if !updateFailurePolicy {
		return fmt.Errorf("webhook is not ready, retry")
	}
	return nil
}
