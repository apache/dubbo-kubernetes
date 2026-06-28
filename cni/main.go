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
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cni"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			if err := runInstall(os.Args[2:]); err != nil {
				exitErr(err)
			}
			return
		case "uninstall":
			if err := runUninstall(os.Args[2:]); err != nil {
				exitErr(err)
			}
			return
		}
	}

	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		exitErr(fmt.Errorf("read stdin: %w", err))
	}

	conf, err := cni.ParseNetConf(stdin)
	if err != nil {
		exitErr(err)
	}
	env := cni.EnvFromOS()
	if env.Command == "VERSION" {
		out, err := cni.Plugin{}.Run(context.Background(), env, conf)
		if err != nil {
			exitErr(err)
		}
		if _, err := os.Stdout.Write(out); err != nil {
			exitErr(fmt.Errorf("write stdout: %w", err))
		}
		return
	}

	plugin := cni.Plugin{
		RuleManager: cni.NewIPTablesRuleManager(conf),
		StateStore:  cni.NewFileStateStore(conf.StateDirectory()),
	}
	if (env.Command == "ADD" || env.Command == "CHECK") && cni.EnvHasKubernetesPod(env) {
		provider, err := cni.NewKubernetesPodInfoProvider(conf.KubeConfigPath())
		if err != nil {
			exitErr(err)
		}
		plugin.PodInfoProvider = provider
	}
	out, err := plugin.Run(context.Background(), env, conf)
	if err != nil {
		exitErr(err)
	}
	if len(out) > 0 {
		if _, err := os.Stdout.Write(out); err != nil {
			exitErr(fmt.Errorf("write stdout: %w", err))
		}
	}
}

func runInstall(args []string) error {
	opts := cni.DefaultInstallerOptions()
	var watch bool
	var interval time.Duration
	flags := installerFlagSet("install", &opts)
	flags.BoolVar(&watch, "watch", false, "keep installer running and refresh service account credentials")
	flags.DurationVar(&interval, "refresh-interval", cni.DefaultInstallerInterval(), "credential refresh interval")
	if err := flags.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if watch {
		return cni.InstallLoop(ctx, opts, interval)
	}
	return cni.Install(ctx, opts)
}

func runUninstall(args []string) error {
	opts := cni.DefaultInstallerOptions()
	flags := installerFlagSet("uninstall", &opts)
	if err := flags.Parse(args); err != nil {
		return err
	}
	return cni.Uninstall(opts)
}

func installerFlagSet(name string, opts *cni.InstallerOptions) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.StringVar(&opts.BinDir, "bin-dir", opts.BinDir, "host CNI binary directory")
	flags.StringVar(&opts.ConfDir, "conf-dir", opts.ConfDir, "host CNI config directory")
	flags.StringVar(&opts.StateDir, "state-dir", opts.StateDir, "host state directory")
	flags.StringVar(&opts.KubeConfigPath, "kubeconfig", opts.KubeConfigPath, "host kubeconfig path used by the CNI plugin")
	flags.StringVar(&opts.TokenFile, "token-file", opts.TokenFile, "host service account token file")
	flags.StringVar(&opts.CAFile, "ca-file", opts.CAFile, "host Kubernetes CA file")
	flags.StringVar(&opts.ServiceAccountTokenPath, "service-account-token", opts.ServiceAccountTokenPath, "mounted service account token source")
	flags.StringVar(&opts.ServiceAccountCAPath, "service-account-ca", opts.ServiceAccountCAPath, "mounted service account CA source")
	flags.StringVar(&opts.APIServer, "api-server", opts.APIServer, "Kubernetes API server URL")
	flags.StringVar(&opts.ManagedLabel, "managed-label", opts.ManagedLabel, "label key that marks mesh-managed Pods")
	flags.StringVar(&opts.ManagedLabelValue, "managed-label-value", opts.ManagedLabelValue, "label value that marks mesh-managed Pods")
	flags.StringVar(&opts.IPTablesPath, "iptables-path", opts.IPTablesPath, "iptables binary used by the CNI plugin")
	flags.StringVar(&opts.IPSetPath, "ipset-path", opts.IPSetPath, "ipset binary used by the CNI plugin")
	flags.IntVar(&opts.GRPCInboundPort, "grpc-inbound-port", opts.GRPCInboundPort, "grpc-inbound inbound port")
	return flags
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
