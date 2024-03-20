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
	"os"
	"path/filepath"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	cmd2 "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/core/cmd/version"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
)

var controlPlaneLog = core.Log.WithName("dubbo-cp")

// newRootCmd represents the base command when called without any subcommands.
func newRootCmd() *cobra.Command {
	args := struct {
		logLevel   string
		outputPath string
		maxSize    int
		maxBackups int
		maxAge     int
	}{}
	cmd := &cobra.Command{
		Use:   "dubbo-cp",
		Short: "Universal Control Plane for Envoy-based Service Mesh",
		Long:  `Universal Control Plane for Envoy-based Service Mesh.`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			level, err := dubbo_log.ParseLogLevel(args.logLevel)
			if err != nil {
				return err
			}

			if args.outputPath != "" {
				output, err := filepath.Abs(args.outputPath)
				if err != nil {
					return err
				}

				fmt.Printf("%s: logs will be stored in %q\n", "dubbo-cp", output)
				core.SetLogger(core.NewLoggerWithRotation(level, output, args.maxSize, args.maxBackups, args.maxAge))
			} else {
				core.SetLogger(core.NewLogger(level))
			}

			// once command line flags have been parsed,
			// avoid printing usage instructions
			cmd.SilenceUsage = true

			return nil
		},
	}

	cmd.SetOut(os.Stdout)

	// root flags
	cmd.PersistentFlags().StringVar(&args.logLevel, "log-level", dubbo_log.InfoLevel.String(), cmd2.UsageOptions("log level", dubbo_log.OffLevel, dubbo_log.InfoLevel, dubbo_log.DebugLevel))
	cmd.PersistentFlags().StringVar(&args.outputPath, "log-output-path", args.outputPath, "path to the file that will be filled with logs. Example: if we set it to /tmp/dubbo.log then after the file is rotated we will have /tmp/dubbo-2021-06-07T09-15-18.265.log")
	cmd.PersistentFlags().IntVar(&args.maxBackups, "log-max-retained-files", 1000, "maximum number of the old log files to retain")
	cmd.PersistentFlags().IntVar(&args.maxSize, "log-max-size", 100, "maximum size in megabytes of a log file before it gets rotated")
	cmd.PersistentFlags().IntVar(&args.maxAge, "log-max-age", 30, "maximum number of days to retain old log files based on the timestamp encoded in their filename")

	// sub-commands
	cmd.AddCommand(newRunCmdWithOpts(cmd2.DefaultRunCmdOpts))
	cmd.AddCommand(version.NewVersionCmd())

	return cmd
}

func DefaultRootCmd() *cobra.Command {
	return newRootCmd()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.maixn(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := DefaultRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
