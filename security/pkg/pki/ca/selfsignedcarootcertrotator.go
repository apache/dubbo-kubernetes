/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ca

import (
	"github.com/apache/dubbo-kubernetes/security/pkg/k8s/controller"
	certutil "github.com/apache/dubbo-kubernetes/security/pkg/util"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
	"math/rand"
	"time"
)

type SelfSignedCARootCertRotatorConfig struct {
	certInspector      certutil.CertUtil
	caStorageNamespace string
	org                string
	rootCertFile       string
	secretName         string
	client             corev1.CoreV1Interface
	CheckInterval      time.Duration
	caCertTTL          time.Duration
	retryInterval      time.Duration
	retryMax           time.Duration
	dualUse            bool
	enableJitter       bool
}

type SelfSignedCARootCertRotator struct {
	caSecretController *controller.CaSecretController
	config             *SelfSignedCARootCertRotatorConfig
	backOffTime        time.Duration
	ca                 *DubboCA
	onRootCertUpdate   func() error
}

func NewSelfSignedCARootCertRotator(config *SelfSignedCARootCertRotatorConfig, ca *DubboCA, onRootCertUpdate func() error) *SelfSignedCARootCertRotator {
	rotator := &SelfSignedCARootCertRotator{
		caSecretController: controller.NewCaSecretController(config.client),
		config:             config,
		ca:                 ca,
		onRootCertUpdate:   onRootCertUpdate,
	}
	if config.enableJitter {
		// Select a back off time in seconds, which is in the range of [0, rotator.config.CheckInterval).
		randSource := rand.NewSource(time.Now().UnixNano())
		randBackOff := rand.New(randSource)
		backOffSeconds := int(time.Duration(randBackOff.Int63n(int64(rotator.config.CheckInterval))).Seconds())
		rotator.backOffTime = time.Duration(backOffSeconds) * time.Second
		klog.Infof("Set up back off time %s to start rotator.", rotator.backOffTime.String())
	} else {
		rotator.backOffTime = time.Duration(0)
	}
	return rotator
}

func (rotator *SelfSignedCARootCertRotator) Run(stopCh chan struct{}) {
	if rotator.config.enableJitter {
		klog.Infof("Jitter is enabled, wait %s before "+
			"starting root cert rotator.", rotator.backOffTime.String())
		select {
		case <-time.After(rotator.backOffTime):
			klog.Infof("Jitter complete, start rotator.")
		case <-stopCh:
			klog.Info("Received stop signal, so stop the root cert rotator.")
			return
		}
	}
	ticker := time.NewTicker(rotator.config.CheckInterval)
	for {
		select {
		case <-ticker.C:
			klog.Info("Check and rotate root cert.")
			// rotator.checkAndRotateRootCert()
		case _, ok := <-stopCh:
			if !ok {
				klog.Info("Received stop signal, so stop the root cert rotator.")
				if ticker != nil {
					ticker.Stop()
				}
				return
			}
		}
	}
}
