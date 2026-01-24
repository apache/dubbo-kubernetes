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

package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/server"
	"github.com/apache/dubbo-kubernetes/pkg/webhooks/validation/controller"
)

func (s *Server) initConfigValidation(args *DubboArgs) error {
	if s.kubeClient == nil {
		return nil
	}
	log.Info("initializing config validator")
	params := server.Options{
		Schemas:      collections.Dubbo,
		DomainSuffix: args.RegistryOptions.KubeOptions.DomainSuffix,
		Mux:          s.httpsMux,
	}
	_, err := server.New(params)
	if err != nil {
		return err
	}
	s.readinessFlags.configValidationReady.Store(true)

	if features.ValidationWebhookConfigName != "" && s.kubeClient != nil {
		s.addStartFunc("validation controller", func(stop <-chan struct{}) error {
			log.Infof("Starting validation controller")
			go controller.NewValidatingWebhookController(
				s.kubeClient, args.Revision, args.Namespace, s.dubbodCertBundleWatcher).Run(stop)
			return nil
		})
	}
	return nil
}
