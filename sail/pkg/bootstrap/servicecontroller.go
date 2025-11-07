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
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/log"

	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
	kubecontroller "github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/kube/controller"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
)

func (s *Server) ServiceController() *aggregate.Controller {
	return s.environment.ServiceDiscovery.(*aggregate.Controller)
}

// initServiceControllers creates and initializes the service controllers
func (s *Server) initServiceControllers(args *SailArgs) error {
	serviceControllers := s.ServiceController()

	registered := sets.New[provider.ID]()
	for _, r := range args.RegistryOptions.Registries {
		serviceRegistry := provider.ID(r)
		if registered.Contains(serviceRegistry) {
			log.Infof("%s registry specified multiple times.", r)
			continue
		}
		registered.Insert(serviceRegistry)
		log.Infof("Adding %s registry adapter", serviceRegistry)
		switch serviceRegistry {
		case provider.Kubernetes:
			if err := s.initKubeRegistry(args); err != nil {
				return err
			}
		default:
			return fmt.Errorf("service registry %s is not supported", r)
		}
	}

	// Defer running of the service controllers.
	s.addStartFunc("service controllers", func(stop <-chan struct{}) error {
		go serviceControllers.Run(stop)
		return nil
	})

	return nil
}

func (s *Server) initKubeRegistry(args *SailArgs) (err error) {
	args.RegistryOptions.KubeOptions.ClusterID = s.clusterID
	args.RegistryOptions.KubeOptions.Revision = args.Revision
	args.RegistryOptions.KubeOptions.KrtDebugger = args.KrtDebugger
	// TODO metrics
	args.RegistryOptions.KubeOptions.XDSUpdater = s.XDSServer
	args.RegistryOptions.KubeOptions.MeshNetworksWatcher = s.environment.NetworksWatcher
	args.RegistryOptions.KubeOptions.MeshWatcher = s.environment.Watcher
	args.RegistryOptions.KubeOptions.SystemNamespace = args.Namespace
	args.RegistryOptions.KubeOptions.MeshServiceController = s.ServiceController()
	kubecontroller.NewMulticluster(args.PodName,
		args.RegistryOptions.KubeOptions,
		s.dubbodCertBundleWatcher,
		args.Revision,
		s.shouldStartNsController(),
		s.environment.ClusterLocal(),
		s.server,
		s.multiclusterController)
	return
}
