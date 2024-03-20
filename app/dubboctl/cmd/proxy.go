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

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
)

import (
	envoy_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"go.uber.org/zap/zapcore"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/app/dubboctl"
	dubbo_cmd "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest"
	"github.com/apache/dubbo-kubernetes/pkg/proxy/envoy"
	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	"github.com/apache/dubbo-kubernetes/pkg/util/template"
)

var runLog = controlPlaneLog.WithName("proxy")

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

func addProxy(opts dubbo_cmd.RunCmdOpts, cmd *cobra.Command) {
	proxyArgs := &dubboctl.ProxyConfig{}
	proxyArgs.SetDefault()
	var proxyResource model.Resource
	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "Commands related to proxy",
		Long:  "Commands help user to generate Ingress and Egress",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("hello world")
			logger.InitCmdSugar(zapcore.AddSync(cmd.OutOrStdout()))
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			proxyTypeMap := map[string]model.ResourceType{
				string(mesh_proto.IngressProxyType): mesh.ZoneIngressType,
				string(mesh_proto.EgressProxyType):  mesh.ZoneEgressType,
			}
			if _, ok := proxyTypeMap[proxyArgs.Dataplane.ProxyType]; !ok {
				return errors.Errorf("invalid proxy type %q", proxyArgs.Dataplane.ProxyType)
			}
			proxyResource, err := readResource(cmd, &proxyArgs.DataplaneRuntime)
			if err != nil {
				runLog.Error(err, "failed to read policy", "proxyType", proxyArgs.Dataplane.ProxyType)
				return err
			}
			if proxyResource != nil {
				if resType := proxyTypeMap[proxyArgs.Dataplane.ProxyType]; resType != proxyResource.Descriptor().Name {
					return errors.Errorf("invalid proxy resource type %q, expected %s",
						proxyResource.Descriptor().Name, resType)
				}
				if proxyArgs.Dataplane.Name != "" || proxyArgs.Dataplane.Mesh != "" {
					return errors.New("--name and --mesh cannot be specified when a dataplane definition is provided, mesh and name will be read from the dataplane definition")
				}

				proxyArgs.Dataplane.Mesh = proxyResource.GetMeta().GetMesh()
				proxyArgs.Dataplane.Name = proxyResource.GetMeta().GetName()
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			// gracefulCtx indicate that the process received a signal to shutdown
			gracefulCtx, _ := opts.SetupSignalHandler()
			_, cancelComponents := context.WithCancel(gracefulCtx)
			opts := envoy.Opts{
				Config:    *proxyArgs,
				Dataplane: rest.From.Resource(proxyResource),
				Stdout:    cmd.OutOrStdout(),
				Stderr:    cmd.OutOrStderr(),
				OnFinish:  cancelComponents,
			}
			envoyVersion, err := envoy.GetEnvoyVersion(opts.Config.DataplaneRuntime.BinaryPath)
			if err != nil {
				return errors.Wrap(err, "failed to get Envoy version")
			}
			runLog.Info("fetched Envoy version", "version", envoyVersion)
			runLog.Info("generating bootstrap configuration")
			bootstrap := envoy_bootstrap_v3.Bootstrap{}

			runLog.Info("received bootstrap configuration", "adminPort", bootstrap.GetAdmin().GetAddress().GetSocketAddress().GetPortValue())
			opts.BootstrapConfig, err = proto.ToYAML(&bootstrap)
			if err != nil {
				return errors.Errorf("could not convert to yaml. %v", err)
			}
			stopComponents := make(chan struct{})
			envoyComponent, err := envoy.New(opts)
			err = envoyComponent.Start(stopComponents)
			if err != nil {
				runLog.Error(err, "error while running Kuma DP")
				return err
			}
			runLog.Info("stopping Dubbo proxy")
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&proxyArgs.Dataplane.Name, "name", proxyArgs.Dataplane.Name, "Name of the Dataplane")
	cmd.PersistentFlags().StringVar(&proxyArgs.Dataplane.Mesh, "mesh", proxyArgs.Dataplane.Mesh, "Mesh that Dataplane belongs to")
	cmd.PersistentFlags().StringVar(&proxyArgs.Dataplane.ProxyType, "proxy-type", "dataplane", `type of the Dataplane ("dataplane", "ingress")`)
	cmd.PersistentFlags().StringVarP(&proxyArgs.DataplaneRuntime.ResourcePath, "dataplane-file", "d", "", "Path to Ingress and Egress template to apply (YAML or JSON)")
	cmd.AddCommand(proxyCmd)
}
