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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
)

func addTemplates(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "List available dubbo source templates",
		SuggestFor: []string{
			"template", "templtaes", "templatse", "remplates",
			"gemplates", "yemplates", "tenplates", "tekplates", "tejplates",
			"temolates", "temllates", "temppates", "tempmates", "tempkates",
			"templstes", "templztes", "templqtes", "templares", "templages", //nolint:misspell
			"templayes", "templatee", "templatea", "templated", "templatew",
		},
		Long: `
NAME
	dubboctl templates - list available dubbo source templates

SYNOPSIS
	dubboctl templates [language] [--json] [-r|--repository]

DESCRIPTION
	List all templates available, optionally for a specific language runtime.

	To specify a URI of a single, specific repository for which templates
	should be displayed, use the --repository flag.

	Installed repositories are by default located at ~/.dubbo/repositories
	($XDG_CONFIG_HOME/.dubbo/repositories).  This can be overridden with
	$DUBBO_REPOSITORIES_PATH.

	To see all available language runtimes, see the 'languages' command.


EXAMPLES

	o Show a list of all available templates grouped by language runtime
	  $ dubboctl templates

	o Show a list of all templates for the Main runtime
	  $ dubboctl templates go

	o Return a list of all template runtimes in JSON output format
	  $ dubboctl templates --json

	o Return Main templates in a specific repository
		$ dubboctl templates go --repository=https://github.com/sjmshsh/dubboctl-samples
`,
		PreRunE: bindEnv("json", "repository"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplates(cmd, args, newClient)
		},
	}
	cmd.Flags().Bool("json", false, "Set output to JSON format. (Env: $DUBBO_JSON)")
	cmd.Flags().StringP("repository", "r", "", "URI to a specific repository to consider ($DUBBO_REPOSITORY)")

	baseCmd.AddCommand(cmd)
}

type templatesConfig struct {
	Repository string // Consider only a specific repository (URI)
	JSON       bool   // output as JSON
}

func runTemplates(cmd *cobra.Command, args []string, newClient ClientFactory) (err error) {
	// Gather config
	cfg, err := newTemplatesConfig(newClient)
	if err != nil {
		return
	}

	// Simple ping to the repo to avoid subsequent errors from http package if it does not exist
	if cfg.Repository != "" {
		res, err := http.Get(cfg.Repository)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode == http.StatusNotFound {
			return dubbo.ErrTemplateRepoDoesNotExist
		}
	}

	// Client which will provide data
	client, done := newClient(
		dubbo.WithRepository(cfg.Repository))
	defer done()

	// For a single language runtime
	if len(args) == 1 {
		templates, err := client.Templates().List(args[0])
		if err != nil {
			return err
		}
		if cfg.JSON {
			s, err := json.MarshalIndent(templates, "", " ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(s))
		} else {
			for _, template := range templates {
				fmt.Fprintln(cmd.OutOrStdout(), template)
			}
		}
		return nil
	} else if len(args) > 1 {
		return errors.New("unexpected extra arguments")
	}
	// All language runtimes
	// ------------
	runtimes, err := client.Runtimes()
	if err != nil {
		return
	}
	if cfg.JSON {
		// Gather into a single data structure for printing as json
		templateMap := make(map[string][]string)
		for _, runtime := range runtimes {
			templates, err := client.Templates().List(runtime)
			if err != nil {
				return err
			}
			templateMap[runtime] = templates
		}
		s, err := json.MarshalIndent(templateMap, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(s))
	} else {
		// print using a formatted writer (sorted)
		builder := strings.Builder{}
		writer := tabwriter.NewWriter(&builder, 0, 0, 3, ' ', 0)
		fmt.Fprint(writer, "LANGUAGE\tTEMPLATE\n")
		for _, runtime := range runtimes {
			templates, err := client.Templates().List(runtime)
			if err != nil {
				return err
			}
			for _, template := range templates {
				fmt.Fprintf(writer, "%v\t%v\n", runtime, template)
			}
		}
		writer.Flush()
		fmt.Fprint(cmd.OutOrStdout(), builder.String())
	}
	return
}

func newTemplatesConfig(newClient ClientFactory) (cfg templatesConfig, err error) {
	cfg = templatesConfig{
		Repository: viper.GetString("repository"),
		JSON:       viper.GetBool("json"),
	}
	return
}
