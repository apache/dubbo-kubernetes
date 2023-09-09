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
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
)

const (
	dockerfileGo   = "dockerTplGo.tpl"
	dockerfileName = "Dockerfile"
)

//go:embed dockerTplGo.tpl
var dockerfileGoTpl string

func addDockerfileGoCmd(baseCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "go",
		Short: "Generate go dockerfile template",
		Long: `
NAME 
	dubboctl docker go - Generate go dockerfile template

SYNOPSIS
	dubboctl docker go [flags]`,
		SuggestFor: []string{"golang", "golnag", "Go", "Golnag"},
		PreRunE:    bindEnv("port", "main", "exe", "version", "base", "path"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerfileGoCmd(cmd, newDockerfileGoConfig(cmd))
		},
	}

	cmd.Flags().IntP("port", "", 0,
		"Port exposed by the container")
	cmd.Flags().StringP("main", "m", "./cmd",
		"")
	cmd.Flags().StringP("exe", "", "dubbogo",
		"The executable name in the built image")
	cmd.Flags().StringP("version", "v", "",
		"golang version")
	cmd.Flags().StringP("base", "b", "scratch",
		"The base image to build the docker image")

	addPathFlag(cmd)

	baseCmd.AddCommand(cmd)
}

type DockerfileGoConfig struct {
	Path       string
	ExposePort int
	Main       string
	Exe        string
	Version    string
	Base       string
}

func newDockerfileGoConfig(cmd *cobra.Command) *DockerfileGoConfig {
	c := &DockerfileGoConfig{
		ExposePort: viper.GetInt("exposePort"),
		Main:       viper.GetString("main"),
		Exe:        viper.GetString("exe"),
		Version:    viper.GetString("version"),
		Base:       viper.GetString("base"),
		Path:       viper.GetString("path"),
	}

	return c
}

func runDockerfileGoCmd(cmd *cobra.Command, cfg *DockerfileGoConfig) error {
	mainFile := cfg.Main
	version := cfg.Version
	root := cfg.Path
	base := cfg.Base
	port := cfg.ExposePort
	exe := cfg.Exe
	if version != "" {
		version = version + "-"
	}

	if len(mainFile) > 0 && !util.FileExists(filepath.Join(root, mainFile)) {
		return fmt.Errorf("file %q not found", mainFile)
	}

	if err := generateGoDockerfile(root, exe, mainFile, base, version, port); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStderr(), "The dockerfile has been generated to the root directory of the project")
	return nil
}

func generateGoDockerfile(root, exe, mainFile, base, version string, port int) error {
	out, err := os.Create(filepath.Join(root, dockerfileName))
	if err != nil {
		return err
	}
	defer out.Close()

	text, err := util.LoadTemplate("", dockerfileGo, dockerfileGoTpl)
	if err != nil {
		return err
	}

	var exeName string
	if len(exe) > 0 {
		exeName = exe
	} else if len(mainFile) > 0 {
		exeName = util.FileNameWithoutExt(filepath.Base(mainFile))
	} else {
		exeName = "dubbogo"
	}

	t := template.Must(template.New("dockerfile").Parse(text))
	if err != nil {
		return err
	}
	return t.Execute(out, Go{
		Chinese:    util.InChina(),
		GoMainFrom: mainFile,
		ExeFile:    exeName,
		BaseImage:  base,
		Port:       port,
		Version:    version,
	})
}

type Go struct {
	Chinese    bool
	GoMainFrom string
	ExeFile    string
	BaseImage  string
	Port       int
	Version    string
}
