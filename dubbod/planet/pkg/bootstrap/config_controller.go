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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	configaggregate "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/config/aggregate"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/config/kube/gateway"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/leaderelection"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/leaderelection/k8sleaderelection/k8sresourcelock"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/url"
	"strings"

	"github.com/apache/dubbo-kubernetes/api/networking/v1alpha3"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/config/kube/crdclient"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/config/kube/file"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/config/memory"
	dubboCredentials "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/credentials"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/credentials/kube"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/adsc"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigSourceAddressScheme string

const (
	File       ConfigSourceAddressScheme = "fs"
	XDS        ConfigSourceAddressScheme = "xds"
	Kubernetes ConfigSourceAddressScheme = "k8s"
)

func (s *Server) makeKubeConfigController(args *PlanetArgs) *crdclient.Client {
	opts := crdclient.Option{
		DomainSuffix: args.RegistryOptions.KubeOptions.DomainSuffix,
		Identifier:   "crd-controller",
		KrtDebugger:  args.KrtDebugger,
	}

	schemas := collections.Planet
	if features.EnableGatewayAPI {
		schemas = collections.PlanetGatewayAPI()
	}
	return crdclient.NewForSchemas(s.kubeClient, opts, schemas)
}

func (s *Server) initK8SConfigStore(args *PlanetArgs) error {
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

		// Use a channel to signal activation of per-revision status writer
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
							// Start informers again. This fixes the case where informers for namespace do not start,
							// as we create them only after acquiring the leader lock
							// Note: stop here should be the overall pilot stop, NOT the leader election stop. We are
							// basically lazy loading the informer, if we stop it when we lose the lock we will never
							// recreate it again.
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

// initConfigSources will process mesh config 'configSources' and initialize
// associated configs.
func (s *Server) initConfigSources(args *PlanetArgs) (err error) {
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
				collections.Planet,
				args.RegistryOptions.KubeOptions,
			)
			if err != nil {
				return err
			}
			s.ConfigStores = append(s.ConfigStores, configController)
			log.Infof("Started File configSource %s", configSource.Address)
		case XDS:
			// XDS config source support (legacy - MCP protocol removed)
			// Note: MCP was a legacy protocol replaced by APIGenerator in Dubbo
			// This XDS config source may not be needed for proxyless mesh
			// TLS settings removed from ConfigSource - use insecure credentials
			// TODO: Implement TLS support when needed
			xdsClient, err := adsc.New(srcAddress.Host, &adsc.ADSConfig{
				InitialDiscoveryRequests: adsc.ConfigInitialRequests(),
				Config: adsc.Config{
					Namespace: args.Namespace,
					Workload:  args.PodName,
					Revision:  args.Revision,
					Meta:      nil,
					GrpcOpts: []grpc.DialOption{
						args.KeepaliveOptions.ConvertToClientOption(),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to dial XDS %s %v", configSource.Address, err)
			}
			store := memory.Make(collections.Planet)
			// TODO: enable namespace filter for memory controller
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

func (s *Server) initConfigController(args *PlanetArgs) error {
	meshGlobalConfig := s.environment.Mesh()
	if len(meshGlobalConfig.ConfigSources) > 0 {
		// XDS config source support (legacy - MCP protocol removed)
		// Note: MCP was a legacy protocol replaced by APIGenerator in Dubbo
		if err := s.initConfigSources(args); err != nil {
			return err
		}
	} else if args.RegistryOptions.FileDir != "" {
		// Local files - should be added even if other options are specified
		configController, err := file.NewController(
			args.RegistryOptions.FileDir,
			args.RegistryOptions.KubeOptions.DomainSuffix,
			collections.Planet,
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

	// Wrap the config controller with a cache.
	aggregateConfigController, err := configaggregate.MakeCache(s.ConfigStores)
	if err != nil {
		return err
	}
	s.configController = aggregateConfigController

	// Create the config store.
	s.environment.ConfigStore = aggregateConfigController

	// Defer starting the controller until after the service is created.
	s.addStartFunc("config controller", func(stop <-chan struct{}) error {
		go s.configController.Run(stop)
		return nil
	})

	return nil
}

// verifyCert verifies given cert against TLS settings like SANs and CRL.
func (s *Server) verifyCert(certs [][]byte, tlsSettings *v1alpha3.ClientTLSSettings) error {
	if len(certs) == 0 {
		return fmt.Errorf("no certificates provided")
	}
	cert, err := x509.ParseCertificate(certs[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	if len(tlsSettings.SubjectAltNames) > 0 {
		sanMatchFound := false
		for _, san := range cert.DNSNames {
			if sanMatchFound {
				break
			}
			for _, name := range tlsSettings.SubjectAltNames {
				if san == name {
					sanMatchFound = true
					break
				}
			}
		}
		if !sanMatchFound {
			return fmt.Errorf("no matching SAN found")
		}
	}

	if len(tlsSettings.CaCrl) > 0 {
		crlData := []byte(strings.TrimSpace(tlsSettings.CaCrl))
		block, _ := pem.Decode(crlData)
		if block != nil {
			crlData = block.Bytes
		}
		crl, err := x509.ParseRevocationList(crlData)
		if err != nil {
			return fmt.Errorf("failed to parse CRL: %w", err)
		}
		for _, revokedCert := range crl.RevokedCertificateEntries {
			if cert.SerialNumber.Cmp(revokedCert.SerialNumber) == 0 {
				return fmt.Errorf("certificate is revoked")
			}
		}
	}

	return nil
}

// getTransportCredentials attempts to create credentials.TransportCredentials from ClientTLSSettings in mesh config
// Implemented only for SIMPLE_TLS mode
// TODO:
//
//	Implement for MUTUAL_TLS/DUBBO_MUTUAL_TLS modes
func (s *Server) getTransportCredentials(args *PlanetArgs, tlsSettings *v1alpha3.ClientTLSSettings) (credentials.TransportCredentials, error) {
	// TODO ValidateTLS

	switch tlsSettings.GetMode() {
	case v1alpha3.ClientTLSSettings_SIMPLE:
		if len(tlsSettings.GetCredentialName()) > 0 {
			rootCert, err := s.getRootCertFromSecret(tlsSettings.GetCredentialName(), args.Namespace)
			if err != nil {
				return nil, err
			}
			tlsSettings.CaCertificates = string(rootCert.Cert)
			tlsSettings.CaCrl = string(rootCert.CRL)
		}
		if tlsSettings.GetInsecureSkipVerify().GetValue() || len(tlsSettings.GetCaCertificates()) == 0 {
			return credentials.NewTLS(&tls.Config{
				ServerName: tlsSettings.GetSni(),
			}), nil
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(tlsSettings.GetCaCertificates())) {
			return nil, fmt.Errorf("failed to add ca certificate from configSource.tlsSettings to pool")
		}
		return credentials.NewTLS(&tls.Config{
			ServerName: tlsSettings.GetSni(),
			RootCAs:    certPool,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return s.verifyCert(rawCerts, tlsSettings)
			},
		}), nil
	default:
		return insecure.NewCredentials(), nil
	}
}

// getRootCertFromSecret fetches a map of keys and values from a secret with name in namespace
func (s *Server) getRootCertFromSecret(name, namespace string) (*dubboCredentials.CertInfo, error) {
	secret, err := s.kubeClient.Kube().CoreV1().Secrets(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get credential with name %v: %v", name, err)
	}
	return kube.ExtractRoot(secret.Data)
}

func (s *Server) checkAndRunNonRevisionLeaderElectionIfRequired(args *PlanetArgs, activateCh chan struct{}) {
	cm, err := s.kubeClient.Kube().CoreV1().ConfigMaps(args.Namespace).Get(context.Background(), leaderelection.GatewayStatusController, v1.GetOptions{})

	if errors.IsNotFound(err) {
		// ConfigMap does not exist, so per-revision leader election should be active
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
				// Non-revision leader election should run, per-revision should be waiting for activation
				s.addTerminatingStartFunc("gateway status", func(stop <-chan struct{}) error {
					secondStop := make(chan struct{})
					// if stop closes, ensure secondStop closes too
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
							// now that we have the leader lock, we can activate the per-revision status writer
							// first close the activateCh channel if it is not already closed
							log.Infof("Activating gateway status writer")
							select {
							case <-activateCh:
								// Channel already closed, do nothing
							default:
								close(activateCh)
							}
							// now end this lease itself
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
	// If annotation missing or holderIdentity is blank, per-revision leader election should be active
	close(activateCh)
}
