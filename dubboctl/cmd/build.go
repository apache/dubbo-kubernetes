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
	"strings"
)

import (
	"github.com/AlecAivazis/survey/v2"

	"github.com/ory/viper"

	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/builders/dockerfile"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/builders/pack"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/util"
)

func addBuild(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Use:        "build",
		Short:      "Build the image for the application",
		Long:       ``,
		SuggestFor: []string{"biuld", "buidl", "built"},
		PreRunE: bindEnv("useDockerfile", "image", "path", "push", "force", "envs",
			"builder-image"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuildCmd(cmd, newClient)
		},
	}

	cmd.Flags().StringP("builder-image", "b", "",
		"Specify a custom builder image for use by the builder other than its default.")
	cmd.Flags().BoolP("useDockerfile", "d", false,
		"Use the dockerfile with the specified path to build")
	cmd.Flags().StringP("image", "i", "",
		"Container image( [registry]/[namespace]/[name]:[tag] )")
	cmd.Flags().BoolP("push", "", true,
		"Whether to push the image to the registry center by the way")
	cmd.Flags().BoolP("force", "f", false,
		"Whether to force build")
	cmd.Flags().StringArrayP("envs", "e", []string{},
		"environment variable for an application, KEY=VALUE format")
	addPathFlag(cmd)
	baseCmd.AddCommand(cmd)
}

func runBuildCmd(cmd *cobra.Command, newClient ClientFactory) error {
	if err := util.CreatePaths(); err != nil {
		return err
	}
	cfg := newBuildConfig(cmd)
	f, err := dubbo.NewDubbo(cfg.Path)
	if err != nil {
		return err
	}

	cfg, err = cfg.Prompt(f)
	if err != nil {
		return err
	}
	if !f.Initialized() {
		return dubbo.NewErrNotInitialized(f.Root)
	}
	cfg.Configure(f)

	clientOptions, err := cfg.buildclientOptions()
	if err != nil {
		return err
	}
	client, done := newClient(clientOptions...)
	defer done()
	if f.Built() && !cfg.Force {
		fmt.Fprintln(cmd.OutOrStdout(), "The Application is up to date, If you still want to build, use `--force true`")
		return nil
	}
	if f, err = client.Build(cmd.Context(), f); err != nil {
		return err
	}
	if cfg.Push {
		if f, err = client.Push(cmd.Context(), f); err != nil {
			return err
		}
	}

	if err = f.Write(); err != nil {
		return err
	}

	return nil
}

func (c *buildConfig) Prompt(d *dubbo.Dubbo) (*buildConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return c, nil
	}

	if c.Image == "" && d.Image == "" {

		qs := []*survey.Question{
			{
				Name:     "image",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "the container image( [registry]/[namespace]/[name]:[tag] ). For example: docker.io/sjmshsh/testapp:latest",
					Default: c.Image,
				},
			},
		}
		if err = survey.Ask(qs, c); err != nil {
			return c, err
		}
	}
	return c, err
}

func (c buildConfig) buildclientOptions() ([]dubbo.Option, error) {
	var o []dubbo.Option

	if c.UseDockerfile {
		o = append(o, dubbo.WithBuilder(dockerfile.NewBuilder()))
	} else {
		o = append(o, dubbo.WithBuilder(pack.NewBuilder()))
	}

	return o, nil
}

type buildConfig struct {
	Envs          []string
	Force         bool
	UseDockerfile bool
	// Push the resulting image to the registry after building.
	Push bool
	// BuilderImage is the image (name or mapping) to use for building.  Usually
	// set automatically.
	BuilderImage string
	Image        string

	// Path of the application implementation on local disk. Defaults to current
	// working directory of the process.
	Path string
}

func newBuildConfig(cmd *cobra.Command) *buildConfig {
	c := &buildConfig{
		Envs:          viper.GetStringSlice("envs"),
		Force:         viper.GetBool("force"),
		UseDockerfile: viper.GetBool("useDockerfile"),
		Push:          viper.GetBool("push"),
		BuilderImage:  viper.GetString("builder-image"),
		Image:         viper.GetString("image"),
		Path:          viper.GetString("path"),
	}

	var err error
	if c.Envs, err = cmd.Flags().GetStringArray("envs"); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "error reading envs: %v\n", err)
	}
	return c
}

func (c *buildConfig) Configure(f *dubbo.Dubbo) {
	if c.Path == "" {
		root, err := os.Getwd()
		if err != nil {
			return
		}
		f.Root = root
	} else {
		f.Root = c.Path
	}
	if c.BuilderImage != "" {
		f.Build.BuilderImages["pack"] = c.BuilderImage
	}
	if c.Image != "" {
		f.Image = c.Image
	}

	if len(c.Envs) > 0 {
		envs := map[string]string{}
		for _, env := range f.Build.BuildEnvs {
			envs[*env.Name] = *env.Value
		}
		for _, pair := range c.Envs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				envs[parts[0]] = parts[1]
			}
		}
		f.Build.BuildEnvs = make([]dubbo.Env, 0, len(envs))
		for k, v := range envs {
			f.Build.BuildEnvs = append(f.Build.BuildEnvs, dubbo.Env{
				Name:  &k,
				Value: &v,
			})
		}
	}
}
