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

package proxy

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/common"
	"github.com/apache/dubbo-kubernetes/operator/pkg/envoy"
	"io"
	"os"
	"path/filepath"
	"strings"
)

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"go.uber.org/zap/zapcore"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/app/dubboctl"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	dubbo_cmd "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	"github.com/apache/dubbo-kubernetes/pkg/util/template"
)

var runLog = common.ControlPlaneLog.WithName("proxy")

type ResourceType string

func readResource(cmd *cobra.Command, r *dubboctl.DataplaneRuntime) (model.Resource, error) {
	var b []byte
	var err error

	// Load from file first.
	switch r.ResourcePath {
	case "":
		if r.Resource != "" {
			b = []byte(r.Resource)
		}
	case "-":
		if b, err = io.ReadAll(cmd.InOrStdin()); err != nil {
			return nil, err
		}
	default:
		if b, err = os.ReadFile(r.ResourcePath); err != nil {
			return nil, errors.Wrap(err, "error while reading provided file")
		}
	}

	if len(b) == 0 {
		return nil, nil
	}

	b = template.Render(string(b), r.ResourceVars)
	runLog.Info("rendered resource", "resource", string(b))

	res, err := rest.YAML.UnmarshalCore(b)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func writeFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), perm); err != nil {
		return err
	}
	return os.WriteFile(filename, data, perm)
}

func AddProxy(opts dubbo_cmd.RunCmdOpts, cmd *cobra.Command) {
	proxyArgs := DefaultProxyConfig()
	cfg := proxyArgs.Config
	var proxyResource model.Resource
	arg := struct {
		logLevel   string
		outputPath string
		maxSize    int
		maxBackups int
		maxAge     int
	}{}

	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Commands related to proxy",
		Long:  "Commands help user to generate Ingress and Egress",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			level, err := dubbo_log.ParseLogLevel(arg.logLevel)
			if err != nil {
				return err
			}
			proxyArgs.LogLevel = level
			if arg.outputPath != "" {
				output, err := filepath.Abs(arg.outputPath)
				if err != nil {
					return err
				}

				fmt.Printf("%s: logs will be stored in %q\n", "dubbo-dp", output)
				core.SetLogger(core.NewLoggerWithRotation(level, output, arg.maxSize, arg.maxBackups, arg.maxAge))
			} else {
				core.SetLogger(core.NewLogger(level))
			}
			proxyTypeMap := map[string]model.ResourceType{
				string(mesh_proto.IngressProxyType): mesh.ZoneIngressType,
				string(mesh_proto.EgressProxyType):  mesh.ZoneEgressType,
			}
			if _, ok := proxyTypeMap[cfg.Dataplane.ProxyType]; !ok {
				return errors.Errorf("invalid proxy type %q", cfg.Dataplane.ProxyType)
			}
			if cfg.DataplaneRuntime.EnvoyLogLevel == "" {
				cfg.DataplaneRuntime.EnvoyLogLevel = proxyArgs.LogLevel.String()
			}

			proxyResource, err = readResource(cmd, &cfg.DataplaneRuntime)
			if err != nil {
				runLog.Error(err, "failed to read policy", "proxyType", cfg.Dataplane.ProxyType)
				return err
			}
			if proxyResource != nil {
				if resType := proxyTypeMap[cfg.Dataplane.ProxyType]; resType != proxyResource.Descriptor().Name {
					return errors.Errorf("invalid proxy resource type %q, expected %s",
						proxyResource.Descriptor().Name, resType)
				}
				if cfg.Dataplane.Name != "" || cfg.Dataplane.Mesh != "" {
					return errors.New("--name and --mesh cannot be specified when a dataplane definition is provided, mesh and name will be read from the dataplane definition")
				}

				cfg.Dataplane.Mesh = proxyResource.GetMeta().GetMesh()
				cfg.Dataplane.Name = proxyResource.GetMeta().GetName()
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// gracefulCtx indicate that the process received a signal to shutdown
			gracefulCtx, _ := opts.SetupSignalHandler()
			_, cancelComponents := context.WithCancel(gracefulCtx)
			opts := envoy.Opts{
				Config:    *cfg,
				Dataplane: rest.From.Resource(proxyResource),
				Stdout:    cmd.OutOrStdout(),
				Stderr:    cmd.OutOrStderr(),
				OnFinish:  cancelComponents,
			}
			//envoyVersion, err := envoy.GetEnvoyVersion(opts.Config.DataplaneRuntime.BinaryPath)
			//if err != nil {
			//	return errors.Wrap(err, "failed to get Envoy version")
			//}
			//runLog.Info("fetched Envoy version", "version", envoyVersion)
			runLog.Info("generating bootstrap configuration")

			bootstrap, _, err := proxyArgs.BootstrapGenerator(gracefulCtx, opts.Config.ControlPlane.URL, opts.Config, envoy.BootstrapParams{
				Dataplane:           opts.Dataplane,
				DNSPort:             cfg.DNS.EnvoyDNSPort,
				EmptyDNSPort:        cfg.DNS.CoreDNSEmptyPort,
				Workdir:             cfg.DataplaneRuntime.SocketDir,
				AccessLogSocketPath: core_xds.AccessLogSocketName(cfg.DataplaneRuntime.SocketDir, cfg.Dataplane.Name, cfg.Dataplane.Mesh),
				MetricsSocketPath:   core_xds.MetricsHijackerSocketName(cfg.DataplaneRuntime.SocketDir, cfg.Dataplane.Name, cfg.Dataplane.Mesh),
				DynamicMetadata:     proxyArgs.BootstrapDynamicMetadata,
				MetricsCertPath:     cfg.DataplaneRuntime.Metrics.CertPath,
				MetricsKeyPath:      cfg.DataplaneRuntime.Metrics.KeyPath,
			})
			if err != nil {
				return errors.Errorf("Failed to generate Envoy bootstrap config. %v", err)
			}
			runLog.Info("received bootstrap configuration", "adminPort", bootstrap.GetAdmin().GetAddress().GetSocketAddress().GetPortValue())
			opts.BootstrapConfig, err = proto.ToYAML(bootstrap)
			if err != nil {
				return errors.Errorf("could not convert to yaml. %v", err)
			}
			opts.AdminPort = bootstrap.GetAdmin().GetAddress().GetSocketAddress().GetPortValue()

			stopComponents := make(chan struct{})
			envoyComponent, err := envoy.New(opts)
			err = envoyComponent.Start(stopComponents)
			if err != nil {
				runLog.Error(err, "error while running Dubbo DP")
				return err
			}
			runLog.Info("stopping Dubbo proxy")
			return nil
		},
	}

	// root flags
	cmd.PersistentFlags().StringVar(&arg.logLevel, "log-level", dubbo_log.InfoLevel.String(), UsageOptions("log level", dubbo_log.OffLevel, dubbo_log.InfoLevel, dubbo_log.DebugLevel))
	cmd.PersistentFlags().StringVar(&arg.outputPath, "log-output-path", arg.outputPath, "path to the file that will be filled with logs. Example: if we set it to /tmp/dubbo.log then after the file is rotated we will have /tmp/dubbo-2021-06-07T09-15-18.265.log")
	cmd.PersistentFlags().IntVar(&arg.maxBackups, "log-max-retained-files", 1000, "maximum number of the old log files to retain")
	cmd.PersistentFlags().IntVar(&arg.maxSize, "log-max-size", 100, "maximum size in megabytes of a log file before it gets rotated")
	cmd.PersistentFlags().IntVar(&arg.maxAge, "log-max-age", 30, "maximum number of days to retain old log files based on the timestamp encoded in their filename")
	cmd.PersistentFlags().StringVar(&cfg.ControlPlane.URL, "cp-address", cfg.ControlPlane.URL, "URL of the Control Plane Dataplane Server. Example: https://localhost:5678")
	proxyCmd.PersistentFlags().StringVar(&cfg.Dataplane.Name, "name", cfg.Dataplane.Name, "Name of the Dataplane")
	proxyCmd.PersistentFlags().StringVar(&cfg.Dataplane.Mesh, "mesh", cfg.Dataplane.Mesh, "Mesh that Dataplane belongs to")
	proxyCmd.PersistentFlags().StringVar(&cfg.Dataplane.ProxyType, "proxy-type", "dataplane", `type of the Dataplane ("dataplane", "ingress")`)
	proxyCmd.PersistentFlags().StringVar(&cfg.DataplaneRuntime.ResourcePath, "dataplane-file", "Path to Ingress and Egress template to apply (YAML or JSON)", "data-plane-file")
	cmd.AddCommand(proxyCmd)
}

func UsageOptions(desc string, options ...interface{}) string {
	values := make([]string, 0, len(options))
	for _, option := range options {
		values = append(values, fmt.Sprintf("%v", option))
	}
	return fmt.Sprintf("%s: one of %s", desc, strings.Join(values, "|"))
}
