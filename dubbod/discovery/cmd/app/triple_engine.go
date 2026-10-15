// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/spf13/cobra"
)

func newTripleEngineCommand() *cobra.Command {
	namespace := firstNonEmpty(os.Getenv("POD_NAMESPACE"), "default")
	domainSuffix := firstNonEmpty(os.Getenv("DOMAIN_SUFFIX"), os.Getenv("TRUST_DOMAIN"), constants.DefaultClusterLocalDomain)
	host, port, hostErr := autoDiscoverServiceTarget(os.Environ(), namespace, domainSuffix)
	host = firstNonEmpty(os.Getenv("DUBBO_SERVICE_HOST"), host)
	port = firstIntFromEnv(port, "DUBBO_SERVICE_PORT")
	var target string
	var directory, manifest, bootstrapFile, instanceIP string
	var watch bool
	var listeners []string
	var statusAddress string
	command := &cobra.Command{
		Use:   "triple-engine",
		Short: "Run the embedded Rust Triple xDS engine.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(listeners) == 0 && os.Getenv("DUBBO_TRIPLE_LISTENERS") == "" {
				if target == "" {
					if host == "" {
						if hostErr != nil {
							return hostErr
						}
						return fmt.Errorf("target host is required; set --host or DUBBO_SERVICE_HOST, or expose exactly one Service in the namespace before starting the pod")
					}
					target = net.JoinHostPort(host, strconv.Itoa(port))
				}
				listeners = []string{target}
			}
			if bootstrapFile != "" {
				if instanceIP != "" {
					if err := bootstrap.WaitForInherentTripleBootstrap(cmd.Context(), bootstrapFile, instanceIP); err != nil {
						return err
					}
				}
				if err := os.Setenv(inject.InherentTripleBootstrapEnvName, bootstrapFile); err != nil {
					return err
				}
			}
			if err := bootstrap.ReleaseInherentTripleBinary(directory, manifest); err != nil {
				return err
			}
			executable := filepath.Join(directory, "triple-engine")
			arguments := []string{executable}
			if watch {
				arguments = append(arguments, "--watch")
			}
			for _, listener := range listeners {
				arguments = append(arguments, "--listener", listener)
			}
			if statusAddress != "" {
				arguments = append(arguments, "--status-address", statusAddress)
			}
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			return syscall.Exec(executable, arguments, os.Environ())
		},
	}
	command.Flags().StringVar(&host, "host", host, "Service DNS name")
	command.Flags().IntVar(&port, "port", port, "Service port")
	command.Flags().StringVar(&target, "target", "", "xDS LDS target; defaults to host:port")
	command.Flags().StringVar(&directory, "directory", filepath.Join(os.TempDir(), "dubbo", "triple"), "Workload executable directory")
	command.Flags().StringVar(&manifest, "manifest", "", "Optional manifest for checking the embedded binary")
	command.Flags().StringVar(&bootstrapFile, "bootstrap", os.Getenv(inject.InherentTripleBootstrapEnvName), "Triple xDS bootstrap file")
	command.Flags().StringVar(&instanceIP, "instance-ip", firstNonEmpty(os.Getenv("INSTANCE_IP"), os.Getenv("POD_IP")), "Assigned workload IP for bootstrap validation")
	command.Flags().BoolVar(&watch, "watch", false, "Watch xDS resource updates")
	command.Flags().StringArrayVar(&listeners, "listener", nil, "xDS listener name; may be repeated")
	command.Flags().StringVar(&statusAddress, "status-address", "", "Runtime status server address")
	return command
}
