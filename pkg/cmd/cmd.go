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
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"os/signal"
	"syscall"
)

func AddFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		fmt.Printf("FLAG: --%s=%q\n", flag.Name, flag.Value)
	})
}

func WaitSignalFunc(cancel context.CancelCauseFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	cancel(fmt.Errorf("received signal: %v", sig.String()))
}
