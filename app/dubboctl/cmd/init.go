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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"

	"github.com/AlecAivazis/survey/v2"
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
)

func addInit(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the application in the current directory as a dubbo application",
		Long: `
NAME
	dubboctl init - Initialize the application as a dubbo application

SYNOPSIS
	dubboctl init [flags]
`,
		PreRunE: bindEnv("path", "runtime"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, newClient)
		},
	}

	cmd.Flags().StringP("runtime", "r", "", "Specify the language(runtime) for this project")

	addPathFlag(cmd)
	baseCmd.AddCommand(cmd)
}

type InitConfig struct {
	Path    string // project path
	Runtime string
}

func newInitConfig(cmd *cobra.Command) (c *InitConfig) {
	c = &InitConfig{
		Path:    viper.GetString("path"),
		Runtime: viper.GetString("runtime"),
	}
	return
}

func (c InitConfig) Validate(client *dubbo.Client) (err error) {
	dirName, _ := deriveNameAndAbsolutePathFromPath(c.Path)
	if err = util.ValidateApplicationName(dirName); err != nil {
		return
	}

	if c.Runtime == "" {
		return noRuntimeError(client)
	}
	if c.Runtime != "" &&
		!isValidRuntime(client, c.Runtime) {
		return newInvalidRuntimeError(client, c.Runtime)
	}

	return
}

func (c *InitConfig) Prompt() (*InitConfig, error) {
	var err error

	if !util.InteractiveTerminal() || c.Runtime != "" {
		return c, nil
	}

	qs := []*survey.Question{
		{
			Name: "runtime",
			Prompt: &survey.Input{
				Message: "Specify the language(runtime) for this project([go] or [java] to enjoy the functions of dubbo)",
				Default: c.Runtime,
			},
		},
	}
	if err = survey.Ask(qs, c); err != nil {
		return c, err
	}
	return c, err
}

func runInit(cmd *cobra.Command, newClient ClientFactory) error {
	cfg := newInitConfig(cmd)
	var err error
	cfg, err = cfg.Prompt()
	client, done := newClient()
	defer done()
	err = cfg.Validate(client)
	if err != nil {
		return err
	}
	var name string
	var path string
	if cfg.Path == "." || cfg.Path == "" {
		path, err = os.Getwd()
		if err != nil {
			return err
		}
		name = filepath.Base(path)
	} else {
		path = cfg.Path
		name = filepath.Base(cfg.Path)
	}

	f := &dubbo.Dubbo{}
	f.Name = name
	f.Runtime = cfg.Runtime
	f.Created = time.Now()
	f.Template = "initialization"
	f.Root = path

	if err := dubbo.EnsureRunDataDir(path); err != nil {
		return err
	}
	err = f.Write()
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStderr(), "It has been initialized as a dubbo project\n")
	return nil
}
