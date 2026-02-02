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
	"context"
	"encoding/json"
	"fmt"
	configaggregate "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/aggregate"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/kube/crdclient"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/kube/file"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/kube/gateway"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/config/memory"
	dubboCredentials "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/credentials"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/credentials/kube"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/leaderelection"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/leaderelection/k8sleaderelection/k8sresourcelock"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/adsc"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
)

type ConfigSourceAddressScheme string

const (
	File       ConfigSourceAddressScheme = "fs"
	XDS        ConfigSourceAddressScheme = "xds"
	Kubernetes ConfigSourceAddressScheme = "k8s"
)

func (s *Server) makeKubeConfigController(args *DubboArgs) *crdclient.Client {
	opts := crdclient.Option{
		DomainSuffix: args.RegistryOptions.KubeOptions.DomainSuffix,
		Identifier:   "crd-controller",
		KrtDebugger:  args.KrtDebugger,
	}

	schemas := collections.Dubbo
	if features.EnableGatewayAPI {
		schemas = collections.DubboGatewayAPI()
	}
	return crdclient.NewForSchemas(s.kubeClient, opts, schemas)
}

func (s *Server) initK8SConfigStore(args *DubboArgs) error {
	if s.kubeClient == nil {
		return nil
	}
	configController := s.makeKubeConfigController(args)
	s.ConfigStores = append(s.ConfigStores, configController)

	if features.EnableGatewayAPI {
		if s.statusManager == nil && features.EnableGatewayAPIStatus {
			s.initStatusManager(args)
		}
		args.RegistryOptions.KubeOptions.KrtDebugger = args.KrtDebugger
		gwc := gateway.NewController(s.kubeClient, s.kubeClient.CrdWatcher().WaitForCRD, args.RegistryOptions.KubeOptions, s.XDSServer)
		s.environment.GatewayAPIController = gwc
		s.ConfigStores = append(s.ConfigStores, s.environment.GatewayAPIController)

		activatePerRevisionStatusWriterCh := make(chan struct{})
		s.checkAndRunNonRevisionLeaderElectionIfRequired(args, activatePerRevisionStatusWriterCh)

		s.addTerminatingStartFunc("gateway status", func(stop <-chan struct{}) error {
			leaderelection.
				NewPerRevisionLeaderElection(args.Namespace, args.PodName, leaderelection.GatewayStatusController, args.Revision, s.kubeClient).
				AddRunFunction(func(leaderStop <-chan struct{}) {
					log.Infof("waiting for gateway status writer activation")
					<-activatePerRevisionStatusWriterCh
					log.Infof("Starting gateway status writer for revision: %s", args.Revision)
					gwc.SetStatusWrite(true, s.statusManager)

					// Trigger a push so we can recompute status
					s.XDSServer.ConfigUpdate(&model.PushRequest{
						Full:   true,
						Reason: model.NewReasonStats(model.GlobalUpdate),
						Forced: true,
					})
					<-leaderStop
					log.Infof("Stopping gateway status writer")
					gwc.SetStatusWrite(false, nil)
				}).
				Run(stop)
			return nil
		})
		if features.EnableGatewayAPIDeploymentController {
			s.addTerminatingStartFunc("gateway deployment controller", func(stop <-chan struct{}) error {
				leaderelection.
					NewPerRevisionLeaderElection(args.Namespace, args.PodName, leaderelection.GatewayDeploymentController, args.Revision, s.kubeClient).
					AddRunFunction(func(leaderStop <-chan struct{}) {
						// We can only run this if the Gateway CRD is created
						if s.kubeClient.CrdWatcher().WaitForCRD(gvr.KubernetesGateway, leaderStop) {
							controller := gateway.NewDeploymentController(s.kubeClient, s.clusterID, s.environment,
								s.webhookInfo.getWebhookConfig, s.webhookInfo.addHandler, nil, args.Revision, args.Namespace)
							s.kubeClient.RunAndWait(stop)
							// TODO tag watcher
							controller.Run(leaderStop)
						}
					}).
					Run(stop)
				return nil
			})
		}
	}
	var err error
	s.RWConfigStore, err = configaggregate.MakeWriteableCache(s.ConfigStores, configController)
	if err != nil {
		return err
	}

	s.XDSServer.ConfigUpdate(&model.PushRequest{
		Full:   true,
		Reason: model.NewReasonStats(model.GlobalUpdate),
		Forced: true,
	})
	return nil
}

func (s *Server) initConfigSources(args *DubboArgs) (err error) {
	for _, configSource := range s.environment.Mesh().ConfigSources {
		srcAddress, err := url.Parse(configSource.Address)
		if err != nil {
			return fmt.Errorf("invalid config URL %s %v", configSource.Address, err)
		}
		scheme := ConfigSourceAddressScheme(srcAddress.Scheme)
		switch scheme {
		case File:
			if srcAddress.Path == "" {
				return fmt.Errorf("invalid fs config URL %s, contains no file path", configSource.Address)
			}

			configController, err := file.NewController(
				srcAddress.Path,
				args.RegistryOptions.KubeOptions.DomainSuffix,
				collections.Dubbo,
				args.RegistryOptions.KubeOptions,
			)
			if err != nil {
				return err
			}
			s.ConfigStores = append(s.ConfigStores, configController)
			log.Infof("Started File configSource %s", configSource.Address)
		case XDS:
			// TODO: Implement TLS support when needed
			xdsClient, err := adsc.New(srcAddress.Host, &adsc.ADSConfig{
				InitialDiscoveryRequests: adsc.ConfigInitialRequests(),
				Config: adsc.Config{
					Namespace: args.Namespace,
					Revision:  args.Revision,
					GrpcOpts: []grpc.DialOption{
						args.KeepaliveOptions.ConvertToClientOption(),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to dial XDS %s %v", configSource.Address, err)
			}
			store := memory.Make(collections.Dubbo)
			configController := memory.NewController(store)
			configController.RegisterHasSyncedHandler(xdsClient.HasSynced)
			xdsClient.Store = configController
			err = xdsClient.Run()
			if err != nil {
				return fmt.Errorf("XDS config source: failed running %v", err)
			}
			s.ConfigStores = append(s.ConfigStores, configController)
			log.Infof("Started XDS configSource %s", configSource.Address)
		case Kubernetes:
			if srcAddress.Path == "" || srcAddress.Path == "/" {
				err2 := s.initK8SConfigStore(args)
				if err2 != nil {
					log.Errorf("Error loading k8s: %v", err2)
					return err2
				}
				log.Infof("Started Kubernetes configSource %s", configSource.Address)
			} else {
				log.Infof("Not implemented, ignore: %v", configSource.Address)
				// TODO: handle k8s:// scheme for remote cluster. Use same mechanism as service registry,
				// using the cluster name as key to match a secret.
			}
		default:
			log.Infof("Ignoring unsupported config source: %v", configSource.Address)
		}
	}
	return nil
}

func (s *Server) initConfigController(args *DubboArgs) error {
	meshGlobalConfig := s.environment.Mesh()
	if len(meshGlobalConfig.ConfigSources) > 0 {
		if err := s.initConfigSources(args); err != nil {
			return err
		}
	} else if args.RegistryOptions.FileDir != "" {
		// Local files - should be added even if other options are specified
		configController, err := file.NewController(
			args.RegistryOptions.FileDir,
			args.RegistryOptions.KubeOptions.DomainSuffix,
			collections.Dubbo,
			args.RegistryOptions.KubeOptions,
		)
		if err != nil {
			return err
		}
		s.ConfigStores = append(s.ConfigStores, configController)
	} else {
		err := s.initK8SConfigStore(args)
		if err != nil {
			return err
		}
	}

	aggregateConfigController, err := configaggregate.MakeCache(s.ConfigStores)
	if err != nil {
		return err
	}
	s.configController = aggregateConfigController

	s.environment.ConfigStore = aggregateConfigController

	s.addStartFunc("config controller", func(stop <-chan struct{}) error {
		go s.configController.Run(stop)
		return nil
	})

	return nil
}

func (s *Server) getRootCertFromSecret(name, namespace string) (*dubboCredentials.CertInfo, error) {
	secret, err := s.kubeClient.Kube().CoreV1().Secrets(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get credential with name %v: %v", name, err)
	}
	return kube.ExtractRoot(secret.Data)
}

func (s *Server) checkAndRunNonRevisionLeaderElectionIfRequired(args *DubboArgs, activateCh chan struct{}) {
	cm, err := s.kubeClient.Kube().CoreV1().ConfigMaps(args.Namespace).Get(context.Background(), leaderelection.GatewayStatusController, v1.GetOptions{})

	if errors.IsNotFound(err) {
		close(activateCh)
		return
	}
	leaderAnn, ok := cm.Annotations[k8sresourcelock.LeaderElectionRecordAnnotationKey]
	if ok {
		var leaderInfo struct {
			HolderIdentity string `json:"holderIdentity"`
		}
		if err := json.Unmarshal([]byte(leaderAnn), &leaderInfo); err == nil {
			if leaderInfo.HolderIdentity != "" {
				s.addTerminatingStartFunc("gateway status", func(stop <-chan struct{}) error {
					secondStop := make(chan struct{})
					go func() {
						<-stop
						select {
						case <-secondStop:
						default:
							close(secondStop)
						}
					}()
					leaderelection.
						NewLeaderElection(args.Namespace, args.PodName, leaderelection.GatewayStatusController, args.Revision, s.kubeClient).
						AddRunFunction(func(leaderStop <-chan struct{}) {
							log.Infof("Activating gateway status writer")
							select {
							case <-activateCh:
							default:
								close(activateCh)
							}
							select {
							case <-secondStop:
							default:
								close(secondStop)
							}
						}).
						Run(secondStop)
					return nil
				})
				return
			}
		}
	}
	close(activateCh)
}
