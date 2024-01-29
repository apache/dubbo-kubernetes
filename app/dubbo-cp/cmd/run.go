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
	"fmt"
	"time"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/bootstrap"
	dubbo_cmd "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/defaults"
	"github.com/apache/dubbo-kubernetes/pkg/diagnostics"
	dp_server "github.com/apache/dubbo-kubernetes/pkg/dp-server"
	"github.com/apache/dubbo-kubernetes/pkg/gc"
	"github.com/apache/dubbo-kubernetes/pkg/hds"
	"github.com/apache/dubbo-kubernetes/pkg/intercp"
	"github.com/apache/dubbo-kubernetes/pkg/util/os"
	dubbo_version "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
)

var runLog = controlPlaneLog.WithName("run")

const gracefullyShutdownDuration = 3 * time.Second

// This is the open file limit below which the control plane may not
// reasonably have enough descriptors to accept all its clients.
const minOpenFileLimit = 4096

func newRunCmdWithOpts(opts dubbo_cmd.RunCmdOpts) *cobra.Command {
	args := struct {
		configPath string
	}{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch Control Plane",
		Long:  `Launch Control Plane.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := dubbo_cp.DefaultConfig()
			err := config.Load(args.configPath, &cfg)
			if err != nil {
				runLog.Error(err, "could not load the configuration")
				return err
			}

			// nolint:staticcheck
			if cfg.Mode == config_core.Standalone {
				runLog.Info(`[WARNING] "standalone" mode is deprecated. Changing it to "zone". Set dubbo_MODE to "zone" as "standalone" will be removed in the future.`)
				cfg.Mode = config_core.Zone
			}

			gracefulCtx, ctx := opts.SetupSignalHandler()
			rt, err := bootstrap.Bootstrap(gracefulCtx, cfg)
			if err != nil {
				runLog.Error(err, "unable to set up Control Plane runtime")
				return err
			}
			cfgForDisplay, err := config.ConfigForDisplay(&cfg)
			if err != nil {
				runLog.Error(err, "unable to prepare config for display")
				return err
			}
			cfgBytes, err := config.ToJson(cfgForDisplay)
			if err != nil {
				runLog.Error(err, "unable to convert config to json")
				return err
			}
			runLog.Info(fmt.Sprintf("Current config %s", cfgBytes))
			runLog.Info(fmt.Sprintf("Running in mode `%s`", cfg.Mode))

			if err := os.RaiseFileLimit(); err != nil {
				runLog.Error(err, "unable to raise the open file limit")
			}

			if limit, _ := os.CurrentFileLimit(); limit < minOpenFileLimit {
				runLog.Info("for better performance, raise the open file limit",
					"minimim-open-files", minOpenFileLimit)
			}

			//if err := admin.Setup(rt); err != nil {
			//	runLog.Error(err, "unable to set up admin server")
			//}
			if err := bufman.Setup(rt); err != nil {
				runLog.Error(err, "unable to set up bufman server")
			}
			if err := xds.Setup(rt); err != nil {
				runLog.Error(err, "unable to set up xds server")
				return err
			}
			if err := hds.Setup(rt); err != nil {
				runLog.Error(err, "unable to set up HDS")
				return err
			}
			if err := dp_server.SetupServer(rt); err != nil {
				runLog.Error(err, "unable to set up DP Server")
				return err
			}
			if err := defaults.Setup(rt); err != nil {
				runLog.Error(err, "unable to set up Defaults")
				return err
			}
			if err := diagnostics.SetupServer(rt); err != nil {
				runLog.Error(err, "unable to set up Diagnostics server")
				return err
			}
			if err := gc.Setup(rt); err != nil {
				runLog.Error(err, "unable to set up GC")
				return err
			}
			if err := intercp.Setup(rt); err != nil {
				runLog.Error(err, "unable to set up Control Plane Intercommunication")
				return err
			}

			runLog.Info("starting Control Plane", "version", dubbo_version.Build.Version)
			if err := rt.Start(gracefulCtx.Done()); err != nil {
				runLog.Error(err, "problem running Control Plane")
				return err
			}

			runLog.Info("stop signal received. Waiting 3 seconds for components to stop gracefully...")
			select {
			case <-ctx.Done():
				runLog.Info("all components have stopped")
			case <-time.After(gracefullyShutdownDuration):
				runLog.Info("forcefully stopped")
			}
			return nil
		},
	}
	// flags
	cmd.PersistentFlags().StringVarP(&args.configPath, "config-file", "c", "", "configuration file")
	return cmd
}
