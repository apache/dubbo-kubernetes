//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pixiu

import (
	"fmt"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"os"
	"os/exec"
	"path/filepath"
)

var pixiulog = dubbolog.RegisterScope("gateway", "pixiu gateway debugging")

type Proxy interface {
	Run(<-chan error) error
	Cleanup()
	UpdateConfig(config []byte) error
}

type ProxyConfig struct {
	ConfigPath    string
	ConfigCleanup bool
	BinaryPath    string
}

type pixiu struct {
	ProxyConfig
}

func NewProxy(cfg ProxyConfig) Proxy {
	return &pixiu{
		ProxyConfig: cfg,
	}
}

func (p *pixiu) UpdateConfig(config []byte) error {
	if err := os.WriteFile(p.ConfigPath, config, 0o666); err != nil {
		return fmt.Errorf("failed to write gateway config: %v", err)
	}
	pixiulog.Infof("updated gateway config at %s", p.ConfigPath)
	return nil
}

func (p *pixiu) Cleanup() {
	if p.ConfigCleanup {
		if err := os.Remove(p.ConfigPath); err != nil {
			pixiulog.Warnf("Failed to delete config file %s: %v", p.ConfigPath, err)
		}
	}
}

func (p *pixiu) Run(abort <-chan error) error {
	if err := os.MkdirAll(filepath.Dir(p.ConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	args := []string{"gateway", "start", "-c", p.ConfigPath}

	pixiulog.Infof("Pixiu command: %s %v", p.BinaryPath, args)

	cmd := exec.Command(p.BinaryPath, args...)
	cmd.Env = os.Environ()

	interceptor := dubbolog.GetInterceptor()
	cmd.Stdout = interceptor.Writer()
	cmd.Stderr = interceptor.Writer()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-abort:
		pixiulog.Warnf("Aborting proxy")
		if errKill := cmd.Process.Kill(); errKill != nil {
			pixiulog.Warnf("killing proxy caused an error %v", errKill)
		}
		return err
	case err := <-done:
		return err
	}
}
