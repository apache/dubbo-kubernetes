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
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation"
	"github.com/apache/dubbo-kubernetes/pkg/log"
)

// initActivation starts the KEDA-facing scaler and the policy controller that
// reports what it can actually do.
//
// Both are optional. Without a Kubernetes client there is nothing to watch, and
// without a listen address KEDA has nothing to dial; either way the rest of the
// control plane runs unchanged, because activation is a per-Service opt-in
// rather than a mesh-wide dependency.
func (s *Server) initActivation(args *DubboArgs) error {
	s.activation = activation.NewServer()

	if err := s.activation.Serve(args.ServerOptions.ActivationAddr, s.internalStop); err != nil {
		return err
	}

	if s.kubeClient == nil {
		log.Info("activation policy controller disabled; no kube client")
		return nil
	}

	controller := activation.NewController(s.kubeClient, s.activation.Scaler(), s.activation.Registry())
	s.addStartFunc("activation policy controller", func(stop <-chan struct{}) error {
		go controller.Run(stop)
		return nil
	})

	return nil
}
