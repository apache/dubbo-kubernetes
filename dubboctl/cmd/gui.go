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
	guiapp "github.com/apache/dubbo-kubernetes/dubbod/gui"
	"github.com/spf13/cobra"
	"net/http"
	"net/url"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"
)

type guiArgs struct {
	address string
	path    string
	open    bool
	wait    time.Duration
}

func GuiCmd() *cobra.Command {
	args := &guiArgs{
		address: "http://127.0.0.1:26080",
		path:    "/gui",
		open:    true,
		wait:    30 * time.Second,
	}

	command := &cobra.Command{
		Use:   "gui",
		Short: "Open the embedded dubbod GUI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			guiURL, overviewURL, err := buildGUIURLs(args.address, args.path)
			if err != nil {
				return err
			}

			if err := waitForOverview(overviewURL, args.wait); err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), guiURL)

			if args.open {
				if err := openBrowser(guiURL); err != nil {
					return err
				}
			}

			return nil
		},
	}

	flags := command.Flags()
	flags.StringVar(&args.address, "address", args.address, "Base HTTP address for dubbod")
	flags.StringVar(&args.path, "path", args.path, "GUI base path on the dubbod HTTP address")
	flags.BoolVar(&args.open, "open", args.open, "Open the GUI in the default browser")
	flags.DurationVar(&args.wait, "wait", args.wait, "Maximum time to wait for the GUI overview endpoint")

	return command
}

func buildGUIURLs(address, guiPath string) (string, string, error) {
	baseAddress := strings.TrimSpace(address)
	if baseAddress == "" {
		return "", "", fmt.Errorf("gui address cannot be empty")
	}
	if !strings.Contains(baseAddress, "://") {
		baseAddress = "http://" + baseAddress
	}

	baseURL, err := url.Parse(baseAddress)
	if err != nil {
		return "", "", fmt.Errorf("invalid gui address %q: %w", address, err)
	}

	normalizedPath := guiapp.NormalizeBasePath(guiPath)
	guiURL := *baseURL
	guiURL.Path = path.Join("/", strings.TrimPrefix(baseURL.Path, "/"), strings.TrimPrefix(normalizedPath, "/"))
	if normalizedPath != "/" && !strings.HasSuffix(guiURL.Path, "/") {
		guiURL.Path += "/"
	}

	overviewURL := *baseURL
	overviewURL.Path = path.Join(guiURL.Path, "api/overview")

	return guiURL.String(), overviewURL.String(), nil
}

func waitForOverview(overviewURL string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	deadline := time.Now().Add(timeout)
	for {
		response, err := client.Get(overviewURL)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}

		if timeout <= 0 || time.Now().After(deadline) {
			return fmt.Errorf("gui overview endpoint is unavailable: %s", overviewURL)
		}

		time.Sleep(500 * time.Millisecond)
	}
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
