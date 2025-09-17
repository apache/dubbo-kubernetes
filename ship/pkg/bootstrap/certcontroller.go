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

package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/ship/pkg/features"
	tb "github.com/apache/dubbo-kubernetes/ship/pkg/trustbundle"
	"k8s.io/klog/v2"
)

const (
	defaultCACertPath = "./var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

func (s *Server) updateRootCertAndGenKeyCert() error {
	klog.Infof("update root cert and generate new dns certs")
	caBundle := s.CA.GetCAKeyCertBundle().GetRootCertPem()
	certChain, keyPEM, err := s.CA.GenKeyCert(s.dnsNames, SelfSignedCACertTTL.Get(), false)
	if err != nil {
		return err
	}

	if features.MultiRootMesh {
		// Trigger trust anchor update, this will send PCDS to all sidecars.
		klog.Infof("Update trust anchor with new root cert")
		err = s.workloadTrustBundle.UpdateTrustAnchor(&tb.TrustAnchorUpdate{
			TrustAnchorConfig: tb.TrustAnchorConfig{Certs: []string{string(caBundle)}},
			Source:            tb.SourceDubboCA,
		})
		if err != nil {
			klog.Errorf("failed to update trust anchor from source Dubbo CA, err: %v", err)
			return err
		}
	}

	s.dubbodCertBundleWatcher.SetAndNotify(keyPEM, certChain, caBundle)
	return nil
}
