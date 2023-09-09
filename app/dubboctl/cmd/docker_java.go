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
	dockerfileJava = "dockerTplJava.tpl"
)

//go:embed dockerTplJava.tpl
var dockerfileJavaTpl string

func addDockerfileJavaCmd(baseCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "java",
		Short: "Generate java dockerfile template",
		Long: `
NAME
	dubboctl docker java - Generate java dockerfile template

SYNOPSIS
	dubboctl docker java [flags]
`,
		SuggestFor: []string{"Java", "jvav", "Jvav"},
		PreRunE:    bindEnv("exposePort", "exe", "version", "jar", "path"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerfileJavaCmd(cmd, newDockerfileJavaConfig())
		},
	}

	cmd.Flags().IntP("exposePort", "e", 0,
		"Ports exposed by the container")
	cmd.Flags().StringP("exe", "", "dubbo",
		"the name of the jar int the container (for example: dubbo)")

	cmd.Flags().StringP("version", "v", "8",
		"java version(default 8)")
	cmd.Flags().StringP("jar", "j", "target/*.jar",
		"The location of the jar package. For example: target/dubbo-samples-docker-1.0-SNAPSHOT.jar")

	addPathFlag(cmd)
	baseCmd.AddCommand(cmd)
}

type DockerfileJavaConfig struct {
	ExeName string
	Port    int
	Version string
	Jar     string
	Path    string
}

type Java struct {
	ExeName string
	Port    int
	Version string
	Jar     string
}

func newDockerfileJavaConfig() *DockerfileJavaConfig {
	c := &DockerfileJavaConfig{
		Port:    viper.GetInt("exposePort"),
		ExeName: viper.GetString("exe"),
		Version: viper.GetString("version"),
		Jar:     viper.GetString("jar"),
		Path:    viper.GetString("path"),
	}

	return c
}

func runDockerfileJavaCmd(cmd *cobra.Command, cfg *DockerfileJavaConfig) error {
	exe := cfg.ExeName
	version := cfg.Version
	jar := cfg.Jar
	port := cfg.Port
	root := cfg.Path
	if err := generateJavaDockerfile(root, exe, version, jar, port); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStderr(), "The dockerfile has been generated to the root directory of the project")
	return nil
}

func generateJavaDockerfile(root, exe, version, jar string, port int) error {
	out, err := os.Create(filepath.Join(root, dockerfileName))
	if err != nil {
		return err
	}
	defer out.Close()

	text, err := util.LoadTemplate("", dockerfileJava, dockerfileJavaTpl)
	if err != nil {
		return err
	}

	t := template.Must(template.New("dockerfile").Parse(text))
	if err != nil {
		return err
	}
	return t.Execute(out, Java{
		ExeName: exe,
		Port:    port,
		Version: version,
		Jar:     jar,
	})
}
