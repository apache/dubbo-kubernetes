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

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
)

func addLanguages(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Use:   "languages",
		Short: "List available function language runtimes",
		Long: `
NAME
	{{rootCmdUse}} languages - list available language runtimes.

SYNOPSIS
	{{rootCmdUse}} languages [--json] [-r|--repository]

DESCRIPTION
	List the language runtimes that are currently available.
	This includes embedded (included) language runtimes as well as any installed
	via the 'repositories add' command.

	To specify a URI of a single, specific repository for which languages
	should be displayed, use the --repository flag.

	Installed repositories are by default located at ~/.dubbo/repositories
	($XDG_CONFIG_HOME/.dubbo/repositories).  This can be overridden with
	$DUBBO_REPOSITORIES_PATH.

	To see templates available for a given language, see the 'templates' command.


EXAMPLES

	o Show a list of all available language runtimes
	  $ {{rootCmdUse}} languages

	o Return a list of all language runtimes in JSON
	  $ {{rootCmdUse}} languages --json

	o Return language runtimes in a specific repository
		$ {{rootCmdUse}} languages --repository=https://github.com/sjmshsh/dubboctl-samples
`,
		SuggestFor: []string{
			"language", "runtime", "runtimes", "lnaguages", "languagse",
			"panguages", "manguages", "kanguages", "lsnguages", "lznguages",
		},
		PreRunE: bindEnv("json", "repository"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLanguages(cmd, args, newClient)
		},
	}

	cmd.Flags().BoolP("json", "", false, "Set output to JSON format. ($DUBBO_JSON)")
	cmd.Flags().StringP("repository", "r", "", "URI to a specific repository to consider ($DUBBO_REPOSITORY)")

	baseCmd.AddCommand(cmd)
}

func runLanguages(cmd *cobra.Command, args []string, newClient ClientFactory) (err error) {
	cfg, err := newLanguagesConfig(newClient)
	if err != nil {
		return
	}

	client, done := newClient(
		dubbo.WithRepository(cfg.Repository))
	defer done()

	runtimes, err := client.Runtimes()
	if err != nil {
		return
	}

	if cfg.JSON {
		var s []byte
		s, err = json.MarshalIndent(runtimes, "", "  ")
		if err != nil {
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(s))
	} else {
		for _, runtime := range runtimes {
			fmt.Fprintln(cmd.OutOrStdout(), runtime)
		}
	}
	return
}

type languagesConfig struct {
	Repository string // Consider only a specific repository (URI)
	JSON       bool   // output as JSON
}

func newLanguagesConfig(newClient ClientFactory) (cfg languagesConfig, err error) {
	cfg = languagesConfig{
		Repository: viper.GetString("repository"),
		JSON:       viper.GetBool("json"),
	}

	return
}
