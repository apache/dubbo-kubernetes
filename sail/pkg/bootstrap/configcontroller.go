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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/adsc"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	configaggregate "github.com/apache/dubbo-kubernetes/sail/pkg/config/aggregate"
	"github.com/apache/dubbo-kubernetes/sail/pkg/config/kube/crdclient"
	"github.com/apache/dubbo-kubernetes/sail/pkg/config/kube/file"
	"github.com/apache/dubbo-kubernetes/sail/pkg/config/memory"
	dubboCredentials "github.com/apache/dubbo-kubernetes/sail/pkg/credentials"
	"github.com/apache/dubbo-kubernetes/sail/pkg/credentials/kube"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"istio.io/api/networking/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"net/url"
	"strings"
)

type ConfigSourceAddressScheme string

const (
	File       ConfigSourceAddressScheme = "fs"
	XDS        ConfigSourceAddressScheme = "xds"
	Kubernetes ConfigSourceAddressScheme = "k8s"
)

func (s *Server) initConfigController(args *SailArgs) error {
	meshConfig := s.environment.Mesh()
	if len(meshConfig.ConfigSources) > 0 {
		// Using MCP for config.
		if err := s.initConfigSources(args); err != nil {
			return err
		}
	} else if args.RegistryOptions.FileDir != "" {
		// Local files - should be added even if other options are specified
		configController, err := file.NewController(
			args.RegistryOptions.FileDir,
			args.RegistryOptions.KubeOptions.DomainSuffix,
			collections.Sail,
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

	// TODO ingress controller
	// TODO addTerminatingStartFunc

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

// initConfigSources will process mesh config 'configSources' and initialize
// associated configs.
func (s *Server) initConfigSources(args *SailArgs) (err error) {
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
				collections.Sail,
				args.RegistryOptions.KubeOptions,
			)
			if err != nil {
				return err
			}
			s.ConfigStores = append(s.ConfigStores, configController)
			klog.Infof("Started File configSource %s", configSource.Address)
		case XDS:
			transportCredentials, err := s.getTransportCredentials(args, configSource.TlsSettings)
			if err != nil {
				return fmt.Errorf("failed to read transport credentials from config: %v", err)
			}
			xdsMCP, err := adsc.New(srcAddress.Host, &adsc.ADSConfig{
				InitialDiscoveryRequests: adsc.ConfigInitialRequests(),
				Config: adsc.Config{
					Namespace: args.Namespace,
					Workload:  args.PodName,
					Revision:  "", // TODO
					Meta:      nil,
					GrpcOpts: []grpc.DialOption{
						args.KeepaliveOptions.ConvertToClientOption(),
						grpc.WithTransportCredentials(transportCredentials),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to dial XDS %s %v", configSource.Address, err)
			}
			store := memory.Make(collections.Sail)
			// TODO: enable namespace filter for memory controller
			configController := memory.NewController(store)
			configController.RegisterHasSyncedHandler(xdsMCP.HasSynced)
			xdsMCP.Store = configController
			err = xdsMCP.Run()
			if err != nil {
				return fmt.Errorf("MCP: failed running %v", err)
			}
			s.ConfigStores = append(s.ConfigStores, configController)
			klog.Infof("Started XDS configSource %s", configSource.Address)
		case Kubernetes:
			if srcAddress.Path == "" || srcAddress.Path == "/" {
				err2 := s.initK8SConfigStore(args)
				if err2 != nil {
					klog.Errorf("Error loading k8s: %v", err2)
					return err2
				}
				klog.Infof("Started Kubernetes configSource %s", configSource.Address)
			} else {
				klog.Infof("Not implemented, ignore: %v", configSource.Address)
				// TODO: handle k8s:// scheme for remote cluster. Use same mechanism as service registry,
				// using the cluster name as key to match a secret.
			}
		default:
			klog.Infof("Ignoring unsupported config source: %v", configSource.Address)
		}
	}
	return nil
}

func (s *Server) makeKubeConfigController(args *SailArgs) *crdclient.Client {
	opts := crdclient.Option{
		DomainSuffix: args.RegistryOptions.KubeOptions.DomainSuffix,
		Identifier:   "crd-controller",
		KrtDebugger:  args.KrtDebugger,
	}

	schemas := collections.Sail

	return crdclient.NewForSchemas(s.kubeClient, opts, schemas)
}

func (s *Server) initK8SConfigStore(args *SailArgs) error {
	if s.kubeClient == nil {
		return nil
	}
	configController := s.makeKubeConfigController(args)
	s.ConfigStores = append(s.ConfigStores, configController)
	s.XDSServer.ConfigUpdate(&model.PushRequest{
		Full:   true,
		Reason: model.NewReasonStats(model.GlobalUpdate),
		Forced: true,
	})
	return nil
}

// getTransportCredentials attempts to create credentials.TransportCredentials from ClientTLSSettings in mesh config
// Implemented only for SIMPLE_TLS mode
// TODO:
//
//	Implement for MUTUAL_TLS/DUBBO_MUTUAL_TLS modes
func (s *Server) getTransportCredentials(args *SailArgs, tlsSettings *v1alpha3.ClientTLSSettings) (credentials.TransportCredentials, error) {
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
				ServerName:         tlsSettings.GetSni(),
				InsecureSkipVerify: tlsSettings.GetInsecureSkipVerify().GetValue(), // nolint
			}), nil
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(tlsSettings.GetCaCertificates())) {
			return nil, fmt.Errorf("failed to add ca certificate from configSource.tlsSettings to pool")
		}
		return credentials.NewTLS(&tls.Config{
			ServerName:         tlsSettings.GetSni(),
			InsecureSkipVerify: tlsSettings.GetInsecureSkipVerify().GetValue(), // nolint
			RootCAs:            certPool,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				return s.verifyCert(rawCerts, tlsSettings)
			},
		}), nil
	default:
		return insecure.NewCredentials(), nil
	}
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

// getRootCertFromSecret fetches a map of keys and values from a secret with name in namespace
func (s *Server) getRootCertFromSecret(name, namespace string) (*dubboCredentials.CertInfo, error) {
	secret, err := s.kubeClient.Kube().CoreV1().Secrets(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get credential with name %v: %v", name, err)
	}
	return kube.ExtractRoot(secret.Data)
}
