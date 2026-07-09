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

package cni

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAddsPluginToConflist(t *testing.T) {
	root := t.TempDir()
	opts := installerTestOptions(t, root)
	confPath := filepath.Join(opts.ConfDir, "10-primary.conflist")
	writeTestFile(t, confPath, `{
  "cniVersion": "1.0.0",
  "name": "primary",
  "plugins": [
    {"type": "bridge", "name": "primary"}
  ]
}`)

	if err := Install(context.Background(), opts); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(opts.BinDir, PluginType)); err != nil {
		t.Fatalf("plugin binary not installed: %v", err)
	}
	plugins := readPlugins(t, confPath)
	if len(plugins) != 2 {
		t.Fatalf("plugins length = %d, want 2", len(plugins))
	}
	if plugins[1]["type"] != PluginType {
		t.Fatalf("last plugin type = %v, want %s", plugins[1]["type"], PluginType)
	}
	kubernetes := plugins[1]["kubernetes"].(map[string]any)
	if kubernetes["kubeconfig"] != opts.KubeConfigPath {
		t.Fatalf("kubeconfig = %v, want %s", kubernetes["kubeconfig"], opts.KubeConfigPath)
	}
	if plugins[1]["ipsetPath"] != defaultIPSetPath {
		t.Fatalf("ipsetPath = %v, want %s", plugins[1]["ipsetPath"], defaultIPSetPath)
	}
	if _, err := os.Stat(confPath + installBackupSuffix); err != nil {
		t.Fatalf("backup not written: %v", err)
	}
}

func TestInstallConvertsConfAndUninstallRestores(t *testing.T) {
	root := t.TempDir()
	opts := installerTestOptions(t, root)
	confPath := filepath.Join(opts.ConfDir, "10-primary.conf")
	writeTestFile(t, confPath, `{"cniVersion":"1.0.0","name":"primary","type":"bridge"}`)

	if err := Install(context.Background(), opts); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Fatalf("primary .conf still exists: %v", err)
	}
	conflistPath := filepath.Join(opts.ConfDir, "10-primary.conflist")
	plugins := readPlugins(t, conflistPath)
	if len(plugins) != 2 || plugins[1]["type"] != PluginType {
		t.Fatalf("converted plugins = %#v", plugins)
	}

	if err := Uninstall(opts); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if _, err := os.Stat(confPath); err != nil {
		t.Fatalf("primary .conf not restored: %v", err)
	}
	if _, err := os.Stat(conflistPath); !os.IsNotExist(err) {
		t.Fatalf("converted conflist still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(opts.BinDir, PluginType)); !os.IsNotExist(err) {
		t.Fatalf("plugin binary still exists: %v", err)
	}
}

func TestInstallRequiresPrimaryConfig(t *testing.T) {
	root := t.TempDir()
	opts := installerTestOptions(t, root)
	if err := Install(context.Background(), opts); err == nil {
		t.Fatal("Install() returned nil error")
	}
}

func installerTestOptions(t *testing.T, root string) InstallerOptions {
	t.Helper()
	source := filepath.Join(root, "source-dubbo-cni")
	writeTestFile(t, source, "binary")
	token := filepath.Join(root, "service-account-token")
	ca := filepath.Join(root, "service-account-ca.crt")
	writeTestFile(t, token, "token")
	writeTestFile(t, ca, "ca")
	return InstallerOptions{
		SourceBinary:            source,
		BinDir:                  filepath.Join(root, "bin"),
		ConfDir:                 filepath.Join(root, "net.d"),
		StateDir:                filepath.Join(root, "state"),
		KubeConfigPath:          filepath.Join(root, "net.d", "dubbo-cni-kubeconfig"),
		TokenFile:               filepath.Join(root, "state", "token"),
		CAFile:                  filepath.Join(root, "state", "ca.crt"),
		ServiceAccountTokenPath: token,
		ServiceAccountCAPath:    ca,
		APIServer:               "https://10.0.0.1:443",
	}
}

func writeTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func readPlugins(t *testing.T, path string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var cfg struct {
		Plugins []map[string]any `json:"plugins"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return cfg.Plugins
}
