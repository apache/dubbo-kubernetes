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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
	"github.com/apache/dubbo-kubernetes/cni/pkg/install"
	"github.com/apache/dubbo-kubernetes/cni/pkg/nodeagent"
	"github.com/apache/dubbo-kubernetes/cni/pkg/tunnel"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var (
		kubeconfig        string
		nodeName          string
		nodeIP            string
		systemNamespace   string
		discoveryAddress  string
		cniBinDir         string
		cniNetDir         string
		cniConfName       string
		binarySource      string
		socketPath        string
		statusAddr        string
		outboundAddr      string
		inboundTunnelAddr string
		excludeNamespaces string
		failOpen          bool
		skipCNIInstall    bool
		disableRedirect   bool
	)

	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig (uses in-cluster config when empty)")
	flag.StringVar(&nodeName, "node-name", os.Getenv(cniconstants.EnvNodeName), "Kubernetes Node name")
	flag.StringVar(&nodeIP, "node-ip", os.Getenv("NODE_IP"), "Node IP for DNAT redirection")
	flag.StringVar(&systemNamespace, "system-namespace", firstNonEmpty(os.Getenv(cniconstants.EnvSystemNamespace), "dubbo-system"), "Dubbo control-plane namespace")
	flag.StringVar(&discoveryAddress, "xds-address", os.Getenv(cniconstants.EnvDiscoveryAddress), "Override dubbod xDS server address")
	flag.StringVar(&cniBinDir, "cni-bin-dir", cniconstants.DefaultCNIBinDir, "Host CNI binary directory")
	flag.StringVar(&cniNetDir, "cni-net-dir", cniconstants.DefaultCNINetDir, "Host CNI config directory")
	flag.StringVar(&cniConfName, "cni-conf-name", "", "Optional explicit CNI conflist file name")
	flag.StringVar(&binarySource, "cni-binary-source", cniconstants.DefaultPluginBinarySource, "Source path of dubbo-cni binary")
	flag.StringVar(&socketPath, "socket-path", cniconstants.DefaultSocketPath, "Unix domain socket path for dubbo-cni callbacks")
	flag.StringVar(&statusAddr, "status-addr", fmt.Sprintf(":%d", cniconstants.DefaultStatusPort), "HTTP address for /healthz, /readyz, /debug/pods, /debug/syncz, /metrics")
	flag.StringVar(&outboundAddr, "outbound-addr", fmt.Sprintf(":%d", cniconstants.OutboundCapturePort), "Outbound transparent capture listen address")
	flag.StringVar(&inboundTunnelAddr, "inbound-tunnel-addr", fmt.Sprintf(":%d", cniconstants.InboundTunnelPort), "Inbound mTLS HTTP/2 CONNECT tunnel listen address")
	flag.StringVar(&excludeNamespaces, "exclude-namespaces", "kube-system,dubbo-system", "Comma-separated namespaces excluded from CNI redirection")
	flag.BoolVar(&failOpen, "fail-open", false, "Allow pod startup if CNI callback to dubbo-cni-node fails")
	flag.BoolVar(&skipCNIInstall, "skip-cni-install", false, "Skip copying dubbo-cni binary and patching CNI conflist")
	flag.BoolVar(&disableRedirect, "disable-redirect", false, "Disable netns iptables redirection (xDS + tunnel only)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var installer *install.Installer
	if !skipCNIInstall {
		var excluded []string
		for _, item := range strings.Split(excludeNamespaces, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				excluded = append(excluded, trimmed)
			}
		}
		installer = install.New(install.Config{
			BinarySource:      binarySource,
			CNIBinDir:         cniBinDir,
			CNINetDir:         cniNetDir,
			CNIConfName:       cniConfName,
			SocketPath:        socketPath,
			FailOpen:          failOpen,
			ExcludeNamespaces: excluded,
		})
		if binPath, err := installer.InstallBinary(); err != nil {
			log.Errorf("failed to install dubbo-cni binary: %v", err)
			os.Exit(1)
		} else {
			log.Infof("installed dubbo-cni plugin binary to %s", binPath)
		}
		if confPath, err := installer.InstallConfig(); err != nil {
			log.Errorf("failed to patch host CNI config: %v", err)
			os.Exit(1)
		} else {
			log.Infof("chained dubbo-cni in %s", confPath)
		}
		defer func() {
			if err := installer.Cleanup(); err != nil {
				log.Warnf("failed to cleanup CNI config on shutdown: %v", err)
			}
		}()

		// Periodically reconcile the host CNI conflist in case the primary CNI rewrites it.
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, err := installer.InstallConfig(); err != nil {
						log.Warnf("periodic CNI config reconcile failed: %v", err)
					}
				}
			}
		}()
	}

	kubeClient, err := buildKubeClient(kubeconfig)
	if err != nil {
		log.Errorf("failed to create Kubernetes client: %v", err)
		os.Exit(1)
	}

	agent := nodeagent.New(kubeClient, nodeagent.Config{
		NodeName:         nodeName,
		NodeIP:           nodeIP,
		SystemNamespace:  systemNamespace,
		DiscoveryAddress: discoveryAddress,
		SocketPath:       socketPath,
		StatusAddr:       statusAddr,
		DisableRedirect:  disableRedirect,
		TunnelConfig: tunnel.Config{
			OutboundAddr:      outboundAddr,
			InboundTunnelAddr: inboundTunnelAddr,
			PeerTunnelPort:    cniconstants.InboundTunnelPort,
		},
	})
	if err := agent.Start(ctx); err != nil {
		log.Errorf("failed to start dubbo-cni-node agent: %v", err)
		os.Exit(1)
	}
	log.Infof("dubbo-cni-node started (node=%q status=%s outbound=%s tunnel=%s)",
		nodeName, agent.StatusAddr(), agent.Proxy().OutboundAddr(), agent.Proxy().InboundTunnelAddr())

	<-ctx.Done()
	log.Infof("dubbo-cni-node shutting down")
	if err := agent.Close(); err != nil {
		log.Warnf("error closing dubbo-cni-node: %v", err)
	}
}

func buildKubeClient(kubeconfig string) (kubernetes.Interface, error) {
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
