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

package main

import (
	"encoding/binary"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"math/rand"
	"net"
	"os"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dubbo-dds-client",
		Short: "dubbo dds client",
		Long:  `dubbo dds client`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			core.SetLogger(core.NewLogger(dubbo_log.DebugLevel))
		},
	}
	cmd.AddCommand(newRunCmd())
	return cmd
}

func newRunCmd() *cobra.Command {
	log := core.Log.WithName("dubbo-dds-client").WithName("run")
	args := struct {
		ddsServerAddress string
		dps              int
	}{
		ddsServerAddress: "grpc://localhost:5679",
		dps:              100,
	}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start dds client(s)",
		Long:  `Start dds client(s)`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ipRand := rand.Uint32()
			log.Info("going to start dds clients")
			errCh := make(chan error, 1)
			for i := 0; i < args.dps; i++ {
				id := fmt.Sprintf("defaul.dataplane-%d", i)
				nodeLog := log.WithName("envoy-simulator").WithValues("idx", i, "ID", id)
				nodeLog.Info("creating an dDS client ...")

				go func(i int) {
					buf := make([]byte, 4)
					binary.LittleEndian.PutUint32(buf, ipRand+uint32(i))
					ip := net.IP(buf).String()
					fmt.Println(ip)
				}(i)
			}

			err := <-errCh

			return errors.Wrap(err, "one of dDS clients (Envoy simulators) terminated with an error")
		},
	}
	cmd.PersistentFlags().StringVar(&args.ddsServerAddress, "dds-server-address", args.ddsServerAddress, "address of dDS server")
	cmd.PersistentFlags().IntVar(&args.dps, "dps", args.dps, "number of dataplanes to emulate")
	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
