// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"fmt"
	"os"
	"strings"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/dubbo"
)

func CompleteRuntimeList(cmd *cobra.Command, args []string, toComplete string, client *dubbo.Client) (matches []string, directive cobra.ShellCompDirective) {
	runtimes, err := client.Runtimes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing runtimes for flag completion: %v\n", err)
		return
	}
	for _, runtime := range runtimes {
		if strings.HasPrefix(runtime, toComplete) {
			matches = append(matches, runtime)
		}
	}
	return
}

func CompleteTemplateList(cmd *cobra.Command, args []string, toComplete string, client *dubbo.Client) (matches []string, directive cobra.ShellCompDirective) {
	directive = cobra.ShellCompDirectiveError

	lang, err := cmd.Flags().GetString("language")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot list templates: %v\n", err)
		return
	}
	if lang == "" {
		fmt.Fprintln(os.Stderr, "cannot list templates: language not specified")
		return
	}

	templates, err := client.Templates().List(lang)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot list templates: %v\n", err)
		return
	}

	directive = cobra.ShellCompDirectiveDefault
	for _, t := range templates {
		if strings.HasPrefix(t, toComplete) {
			matches = append(matches, t)
		}
	}

	return
}
