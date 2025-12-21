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

package pixiu

import (
	"fmt"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"os"
	"os/exec"
	"path/filepath"
)

var log = dubbolog.RegisterScope("gateway", "pixiu gateway proxy")

type Proxy interface {
	Run(<-chan error) error
	Drain(skipExit bool) error
	Cleanup()
	UpdateConfig(config []byte) error
}

type ProxyConfig struct {
	ConfigPath    string
	ConfigCleanup bool
	BinaryPath    string
}

type pixiuProxy struct {
	ProxyConfig
}

func NewProxy(cfg ProxyConfig) Proxy {
	return &pixiuProxy{
		ProxyConfig: cfg,
	}
}

func (p *pixiuProxy) Drain(skipExit bool) error {
	return nil
}

func (p *pixiuProxy) UpdateConfig(config []byte) error {
	if err := os.WriteFile(p.ConfigPath, config, 0o666); err != nil {
		return fmt.Errorf("failed to write gateway config: %v", err)
	}
	log.Infof("updated gateway config at %s", p.ConfigPath)
	return nil
}

func (p *pixiuProxy) Cleanup() {
	if p.ConfigCleanup {
		if err := os.Remove(p.ConfigPath); err != nil {
			log.Warnf("Failed to delete config file %s: %v", p.ConfigPath, err)
		}
	}
}

func (p *pixiuProxy) Run(abort <-chan error) error {
	if err := os.MkdirAll(filepath.Dir(p.ConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	args := []string{
		"gateway",
		"start",
		"-c", p.ConfigPath,
	}

	log.Infof("pixiu command: %s %v", p.BinaryPath, args)

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
		log.Warnf("aborting gateway")
		if errKill := cmd.Process.Kill(); errKill != nil {
			log.Warnf("killing gateway caused an error %v", errKill)
		}
		return err
	case err := <-done:
		return err
	}
}
