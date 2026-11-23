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

package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
)

func TestInstallerBinaryAndConflistLifecycle(t *testing.T) {
	tmp := t.TempDir()
	srcBin := filepath.Join(tmp, "src-dubbo-cni")
	if err := os.WriteFile(srcBin, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(tmp, "bin")
	netDir := filepath.Join(tmp, "net.d")
	if err := os.MkdirAll(netDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conflistPath := filepath.Join(netDir, "10-kindnet.conflist")
	initial := `{
  "cniVersion": "1.0.0",
  "name": "kindnet",
  "plugins": [
    {"type": "ptp", "ipam": {"type": "host-local"}},
    {"type": "portmap"}
  ]
}`
	if err := os.WriteFile(conflistPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := New(Config{
		BinarySource: srcBin,
		CNIBinDir:    binDir,
		CNINetDir:    netDir,
		SocketPath:   "/var/run/dubbo-cni/cni.sock",
	})

	dstBin, err := inst.InstallBinary()
	if err != nil {
		t.Fatalf("InstallBinary() error = %v", err)
	}
	if _, err := os.Stat(dstBin); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}

	// Idempotent install twice.
	for attempt := 0; attempt < 2; attempt++ {
		patched, err := inst.InstallConfig()
		if err != nil {
			t.Fatalf("InstallConfig() attempt %d error = %v", attempt, err)
		}
		if patched != conflistPath {
			t.Fatalf("patched = %q, want %q", patched, conflistPath)
		}
	}

	plugins := readPlugins(t, conflistPath)
	if len(plugins) != 3 || plugins[2]["type"] != cniconstants.PluginName {
		t.Fatalf("unexpected plugins after install: %+v", plugins)
	}

	if err := inst.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	plugins = readPlugins(t, conflistPath)
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins after cleanup, got %+v", plugins)
	}
}

func TestInstallerUpgradesSingleConfToConflist(t *testing.T) {
	netDir := t.TempDir()
	confPath := filepath.Join(netDir, "10-bridge.conf")
	if err := os.WriteFile(confPath, []byte(`{"cniVersion":"0.4.0","name":"bridge","type":"bridge"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	inst := New(Config{CNINetDir: netDir})
	outPath, err := inst.InstallConfig()
	if err != nil {
		t.Fatalf("InstallConfig(.conf) error = %v", err)
	}
	wantPath := confPath + "list"
	if outPath != wantPath {
		t.Fatalf("outPath = %q, want %q", outPath, wantPath)
	}
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Fatalf("old .conf should have been replaced by .conflist")
	}
	plugins := readPlugins(t, wantPath)
	if len(plugins) != 2 || plugins[0]["type"] != "bridge" || plugins[1]["type"] != cniconstants.PluginName {
		t.Fatalf("unexpected upgraded plugins: %+v", plugins)
	}
}

func readPlugins(t *testing.T, path string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Plugins []map[string]any `json:"plugins"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Plugins
}
