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

package envoy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/Masterminds/semver/v3"

	envoy_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/app/dubboctl"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest"
	command_utils "github.com/apache/dubbo-kubernetes/pkg/proxy/command"
	"github.com/apache/dubbo-kubernetes/pkg/util/files"
	"github.com/apache/dubbo-kubernetes/pkg/xds/bootstrap/types"
)

var runLog = core.Log.WithName("dubbo-proxy").WithName("run").WithName("envoy")

type BootstrapConfigFactoryFunc func(ctx context.Context, url string, cfg dubboctl.Config, params BootstrapParams) (*envoy_bootstrap_v3.Bootstrap, *types.DubboSidecarConfiguration, error)

type BootstrapParams struct {
	Dataplane           rest.Resource
	DNSPort             uint32
	EmptyDNSPort        uint32
	EnvoyVersion        EnvoyVersion
	DynamicMetadata     map[string]string
	Workdir             string
	MetricsSocketPath   string
	AccessLogSocketPath string
	MetricsCertPath     string
	MetricsKeyPath      string
}

type EnvoyVersion struct {
	Build             string
	Version           string
	DubboDpCompatible bool
}

type Opts struct {
	Config          dubboctl.Config
	BootstrapConfig []byte
	AdminPort       uint32
	Dataplane       rest.Resource
	Stdout          io.Writer
	Stderr          io.Writer
	OnFinish        func()
}

type Envoy struct {
	opts Opts

	wg sync.WaitGroup
}

func New(opts Opts) (*Envoy, error) {
	if opts.OnFinish == nil {
		opts.OnFinish = func() {}
	}
	return &Envoy{opts: opts}, nil
}

func GenerateBootstrapFile(cfg dubboctl.DataplaneRuntime, config []byte) (string, error) {
	configFile := filepath.Join(cfg.ConfigDir, "bootstrap.yaml")
	if err := writeFile(configFile, config, 0o600); err != nil {
		return "", errors.Wrap(err, "failed to persist Envoy bootstrap config on disk")
	}
	return configFile, nil
}

func writeFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filename, data, perm)
}

func (e *Envoy) Start(stop <-chan struct{}) error {
	e.wg.Add(1)
	// Component should only be considered done after Envoy exists.
	// Otherwise, we may not propagate SIGTERM on time.
	defer func() {
		e.wg.Done()
		e.opts.OnFinish()
	}()

	configFile, err := GenerateBootstrapFile(e.opts.Config.DataplaneRuntime, e.opts.BootstrapConfig)
	if err != nil {
		return err
	}
	runLog.Info("bootstrap configuration saved to a file", "file", configFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	binaryPathConfig := e.opts.Config.DataplaneRuntime.BinaryPath
	resolvedPath, err := lookupEnvoyPath(binaryPathConfig)
	if err != nil {
		return err
	}

	args := []string{
		"--config-path", configFile,
		"--drain-time-s",
		fmt.Sprintf("%d", e.opts.Config.Dataplane.DrainTime.Duration/time.Second),
		// "hot restart" (enabled by default) requires each Envoy instance to have
		// `--base-id <uint32_t>` argument.
		// it is not possible to start multiple Envoy instances on the same Linux machine
		// without `--base-id <uint32_t>` set.
		// although we could come up with a solution how to generate `--base-id <uint32_t>`
		// automatically, it is not strictly necessary since we're not using "hot restart"
		// and we don't expect users to do "hot restart" manually.
		// so, let's turn it off to simplify getting started experience.
		"--disable-hot-restart",
		"--log-level", e.opts.Config.DataplaneRuntime.EnvoyLogLevel,
	}

	if e.opts.Config.DataplaneRuntime.EnvoyComponentLogLevel != "" {
		args = append(args, "--component-log-level", e.opts.Config.DataplaneRuntime.EnvoyComponentLogLevel)
	}

	// If the concurrency is explicit, use that. On Linux, users
	// can also implicitly set concurrency using cpusets.
	if e.opts.Config.DataplaneRuntime.Concurrency > 0 {
		args = append(args,
			"--concurrency",
			strconv.FormatUint(uint64(e.opts.Config.DataplaneRuntime.Concurrency), 10),
		)
	} else if runtime.GOOS == "linux" {
		// The `--cpuset-threads` flag is still present on
		// non-Linux, but emits a warning that we might as well
		// avoid.
		args = append(args, "--cpuset-threads")
	}

	command := command_utils.BuildCommand(ctx, e.opts.Stdout, e.opts.Stderr, resolvedPath, args...)

	runLog.Info("starting Envoy", "path", resolvedPath, "arguments", args)
	if err := command.Start(); err != nil {
		runLog.Error(err, "envoy executable failed", "path", resolvedPath, "arguments", args)
		return err
	}
	go func() {
		<-stop
		runLog.Info("stopping Envoy")
		cancel()
	}()
	err = command.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		runLog.Error(err, "Envoy terminated with an error")
		return err
	}
	runLog.Info("Envoy terminated successfully")
	return nil
}

func lookupEnvoyPath(configuredPath string) (string, error) {
	return files.LookupBinaryPath(
		files.LookupInPath(configuredPath),
		files.LookupInCurrentDirectory("envoy"),
		files.LookupNextToCurrentExecutable("envoy"),
	)
}

func GetEnvoyVersion(binaryPath string) (*EnvoyVersion, error) {
	resolvedPath, err := lookupEnvoyPath(binaryPath)
	if err != nil {
		return nil, err
	}
	arg := "--version"
	command := exec.Command(resolvedPath, arg)
	output, err := command.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s with arguments %q", resolvedPath, arg)
	}
	build := strings.ReplaceAll(string(output), "\r\n", "\n")
	build = strings.Trim(build, "\n")
	build = regexp.MustCompile(`version:(.*)`).FindString(build)
	build = strings.Trim(build, "version:")
	build = strings.Trim(build, " ")

	parts := strings.Split(build, "/")
	if len(parts) != 5 { // revision/build_version_number/revision_status/build_type/ssl_version
		return nil, errors.Errorf("wrong Envoy build format: %s", build)
	}
	return &EnvoyVersion{
		Build:   build,
		Version: parts[1],
	}, nil
}

func VersionCompatible(expectedVersion string, envoyVersion string) (bool, error) {
	ver, err := semver.NewVersion(envoyVersion)
	if err != nil {
		return false, errors.Wrapf(err, "unable to parse envoy version %s", envoyVersion)
	}

	constraint, err := semver.NewConstraint(expectedVersion)
	if err != nil {
		// Programmer error
		panic(errors.Wrapf(err, "Invalid envoy compatibility constraint %s", expectedVersion))
	}

	return constraint.Check(ver), nil
}
