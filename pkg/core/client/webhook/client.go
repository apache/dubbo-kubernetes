// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import (
	"context"
	"reflect"

	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	admissionregistrationV1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Client interface {
	UpdateWebhookConfig(options *dubbo_cp.Config, CertPem string)
	GetNamespaceLabels(namespace string) map[string]string
	GetKubClient() kubernetes.Interface
}

type ClientImpl struct {
	kubeClient kubernetes.Interface
}

func NewClient(kubeClient kubernetes.Interface) Client {
	return &ClientImpl{
		kubeClient: kubeClient,
	}
}

func (c *ClientImpl) GetNamespaceLabels(namespace string) map[string]string {
	ns, err := c.kubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to validate token. " + err.Error())
		return map[string]string{}
	}
	if ns.Labels != nil {
		return ns.Labels
	}
	return map[string]string{}
}

func (c *ClientImpl) UpdateWebhookConfig(options *dubbo_cp.Config, CertPem string) {
	path := "/mutating-services"
	failurePolicy := admissionregistrationV1.Ignore
	sideEffects := admissionregistrationV1.SideEffectClassNone
	bundle := CertPem
	mwConfig, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), "dubbo-cp", metav1.GetOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Webhook] Unable to find dubbo-cp webhook config. Will create. " + err.Error())
		mwConfig = &admissionregistrationV1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dubbo-cp",
			},
			Webhooks: []admissionregistrationV1.MutatingWebhook{
				{
					Name: "dubbo-cp" + ".k8s.io",
					ClientConfig: admissionregistrationV1.WebhookClientConfig{
						Service: &admissionregistrationV1.ServiceReference{
							Name:      options.KubeConfig.ServiceName,
							Namespace: options.KubeConfig.Namespace,
							Port:      &options.Security.WebhookPort,
							Path:      &path,
						},
						CABundle: []byte(bundle),
					},
					FailurePolicy: &failurePolicy,
					Rules: []admissionregistrationV1.RuleWithOperations{
						{
							Operations: []admissionregistrationV1.OperationType{
								admissionregistrationV1.Create,
							},
							Rule: admissionregistrationV1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods"},
							},
						},
					},
					// TODO add it or not?
					//NamespaceSelector: &metav1.LabelSelector{
					//	MatchLabels: map[string]string{
					//		"dubbo-injection": "enabled",
					//	},
					//},
					//ObjectSelector: &metav1.LabelSelector{
					//	MatchLabels: map[string]string{
					//		"dubbo-injection": "enabled",
					//	},
					//},
					SideEffects:             &sideEffects,
					AdmissionReviewVersions: []string{"v1"},
				},
			},
		}

		_, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), mwConfig, metav1.CreateOptions{})
		if err != nil {
			logger.Sugar().Warnf("[Webhook] Failed to create webhook config. " + err.Error())
		} else {
			logger.Sugar().Info("[Webhook] Create webhook config success.")
		}
		return
	}

	if reflect.DeepEqual(mwConfig.Webhooks[0].ClientConfig.CABundle, []byte(bundle)) {
		logger.Sugar().Info("[Webhook] Ignore override webhook config. Cause: Already exist.")
		return
	}

	mwConfig.Webhooks[0].ClientConfig.CABundle = []byte(bundle)
	_, err = c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.TODO(), mwConfig, metav1.UpdateOptions{})
	if err != nil {
		logger.Sugar().Warnf("[Webhook] Failed to update webhook config. " + err.Error())
	} else {
		logger.Sugar().Info("[Webhook] Update webhook config success.")
	}
}

func (c *ClientImpl) GetKubClient() kubernetes.Interface {
	return c.kubeClient
}
