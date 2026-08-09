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

package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// launchEnvVar gates the console binary: it refuses to boot unless dubboctl
// started it, so the console is never exposed as a standalone service.
const launchEnvVar = "DUBBOCTL_GUI_LAUNCH"

const defaultConsoleBinary = "dubbod-console"

var osExecutable = os.Executable

type guiArgs struct {
	binary     string
	listen     string
	basePath   string
	kubeconfig string
	contexts   []string
	endpoints  []string
	open       bool
	wait       time.Duration
}

func GuiCmd() *cobra.Command {
	args := &guiArgs{
		binary:   defaultConsoleBinary,
		listen:   "127.0.0.1:8080",
		basePath: "/",
		open:     true,
		wait:     30 * time.Second,
	}

	command := &cobra.Command{
		Use:   "gui [-- CONSOLE FLAGS]",
		Short: "Start the dubbod console and open it in a browser",
		Long: "Start the dubbod console and open it in a browser.\n\n" +
			"The console is a separate binary that aggregates the management API of every\n" +
			"discovered control plane. Put dubbod-console next to dubboctl or in PATH.\n" +
			"The console only runs when started through this command.\n" +
			"Arguments after -- are passed to the console unchanged.",
		RunE: func(cmd *cobra.Command, extraArgs []string) error {
			binary, err := findConsoleBinary(args.binary)
			if err != nil {
				return err
			}

			consoleURL, healthURL, err := buildGUIURLs(args.listen, args.basePath)
			if err != nil {
				return err
			}

			console := exec.Command(binary, args.consoleArgs(extraArgs)...)
			console.Env = append(os.Environ(), launchEnvVar+"=1")
			console.Stdout = cmd.OutOrStdout()
			console.Stderr = cmd.ErrOrStderr()
			if err := console.Start(); err != nil {
				return fmt.Errorf("failed to start console: %w", err)
			}

			exited := make(chan error, 1)
			go func() { exited <- console.Wait() }()

			if err := waitForConsole(healthURL, args.wait, exited); err != nil {
				terminate(console)
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), consoleURL)

			if args.open {
				if err := openBrowser(consoleURL); err != nil {
					terminate(console)
					return err
				}
			}

			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(signals)

			select {
			case err := <-exited:
				return err
			case <-signals:
				terminate(console)
				<-exited
				return nil
			}
		},
	}

	flags := command.Flags()
	flags.StringVar(&args.binary, "binary", args.binary, "Console binary name or path")
	flags.StringVar(&args.listen, "listen", args.listen, "Address the console listens on")
	flags.StringVar(&args.basePath, "base-path", args.basePath, "Console HTTP base path")
	flags.StringVar(&args.kubeconfig, "kubeconfig", args.kubeconfig, "Kubeconfig used to discover control planes; the default kubeconfig is used when empty")
	flags.StringSliceVar(&args.contexts, "context", args.contexts, "Kubeconfig context to discover; repeatable for cross-control-plane views")
	flags.StringSliceVar(&args.endpoints, "endpoint", args.endpoints, "Static control-plane endpoint as name=http://host:port; repeatable")
	flags.BoolVar(&args.open, "open", args.open, "Open the console in the default browser")
	flags.DurationVar(&args.wait, "wait", args.wait, "Maximum time to wait for the console to become healthy")

	return command
}

// consoleArgs maps the dubboctl flags onto the console's own flag set. Discovery
// falls back to the default kubeconfig so `dubboctl gui` works with no flags.
func (a *guiArgs) consoleArgs(extraArgs []string) []string {
	consoleArgs := []string{
		"--listen", a.listen,
		"--base-path", a.basePath,
	}

	if a.kubeconfig != "" {
		consoleArgs = append(consoleArgs, "--kubeconfig", a.kubeconfig)
	}
	for _, context := range a.contexts {
		consoleArgs = append(consoleArgs, "--context", context)
	}
	for _, endpoint := range a.endpoints {
		consoleArgs = append(consoleArgs, "--endpoint", endpoint)
	}
	if a.kubeconfig == "" && len(a.contexts) == 0 && len(a.endpoints) == 0 {
		consoleArgs = append(consoleArgs, "--discover-kubeconfig")
	}

	return append(consoleArgs, extraArgs...)
}

func buildGUIURLs(listen, basePath string) (string, string, error) {
	address := strings.TrimSpace(listen)
	if address == "" {
		return "", "", fmt.Errorf("console listen address cannot be empty")
	}
	if !strings.Contains(address, "://") {
		address = "http://" + normalizeListenHost(address)
	}

	baseURL, err := url.Parse(address)
	if err != nil {
		return "", "", fmt.Errorf("invalid console listen address %q: %w", listen, err)
	}

	consoleURL := *baseURL
	consoleURL.Path = path.Join("/", strings.TrimPrefix(basePath, "/"))
	if consoleURL.Path != "/" {
		consoleURL.Path += "/"
	}

	healthURL := *baseURL
	healthURL.Path = path.Join(consoleURL.Path, "readyz")

	return consoleURL.String(), healthURL.String(), nil
}

func findConsoleBinary(name string) (string, error) {
	if binary, err := exec.LookPath(name); err == nil {
		return binary, nil
	}

	if filepath.Base(name) == name {
		executable, err := osExecutable()
		if err == nil {
			candidate := filepath.Join(filepath.Dir(executable), consoleBinaryFilename(name))
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf(
		"console binary %q not found; install it next to dubboctl, add it to PATH, or pass --binary",
		name,
	)
}

func consoleBinaryFilename(name string) string {
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		return name + ".exe"
	}
	return name
}

// normalizeListenHost turns a wildcard listen address into something dialable,
// so `--listen :8080` still yields a URL the browser can open.
func normalizeListenHost(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func waitForConsole(healthURL string, timeout time.Duration, exited <-chan error) error {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	deadline := time.Now().Add(timeout)
	for {
		select {
		case err := <-exited:
			if err != nil {
				return fmt.Errorf("console exited before becoming healthy: %w", err)
			}
			return fmt.Errorf("console exited before becoming healthy")
		default:
		}

		response, err := client.Get(healthURL)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}

		if timeout <= 0 || time.Now().After(deadline) {
			return fmt.Errorf("console health endpoint is unavailable: %s", healthURL)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func terminate(console *exec.Cmd) {
	if console.Process == nil {
		return
	}
	_ = console.Process.Signal(syscall.SIGTERM)
}

func openBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("cmd", "/c", "start", "", target)
	default:
		command = exec.Command("xdg-open", target)
	}

	if err := command.Start(); err != nil {
		return fmt.Errorf("failed to open browser for %s: %w", target, err)
	}

	return nil
}
