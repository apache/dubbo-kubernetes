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

package nodeagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
)

const (
	defaultCNIBinDir               = "/opt/cni/bin"
	defaultCNIConfDir              = "/etc/cni/net.d"
	defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultServiceAccountCAPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	defaultInstallerInterval       = time.Minute
	defaultKubeConfigFileName      = "dubbo-cni-kubeconfig"
	installBackupSuffix            = ".dubbo-cni.bak"
)

type InstallerOptions struct {
	SourceBinary            string
	BinDir                  string
	ConfDir                 string
	StateDir                string
	KubeConfigPath          string
	TokenFile               string
	CAFile                  string
	ServiceAccountTokenPath string
	ServiceAccountCAPath    string
	APIServer               string
	ManagedLabel            string
	ManagedLabelValue       string
	IPTablesPath            string
	IPSetPath               string
	GRPCInboundPort         int
}

func DefaultInstallerOptions() InstallerOptions {
	return InstallerOptions{
		BinDir:                  defaultCNIBinDir,
		ConfDir:                 defaultCNIConfDir,
		StateDir:                defaultStateDir,
		ServiceAccountTokenPath: defaultServiceAccountTokenPath,
		ServiceAccountCAPath:    defaultServiceAccountCAPath,
		ManagedLabel:            inject.ProxylessManagedLabel,
		ManagedLabelValue:       inject.ProxylessManagedLabelValue,
		IPTablesPath:            defaultIPTablesPath,
		IPSetPath:               defaultIPSetPath,
		GRPCInboundPort:         inject.ProxylessGRPCInboundPort,
	}
}

func DefaultInstallerInterval() time.Duration {
	return defaultInstallerInterval
}

func (o *InstallerOptions) applyDefaults(requireAPIServer bool) error {
	defaults := DefaultInstallerOptions()
	if o.SourceBinary == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("find current executable: %w", err)
		}
		o.SourceBinary = exe
	}
	if o.BinDir == "" {
		o.BinDir = defaults.BinDir
	}
	if o.ConfDir == "" {
		o.ConfDir = defaults.ConfDir
	}
	if o.StateDir == "" {
		o.StateDir = defaults.StateDir
	}
	if o.KubeConfigPath == "" {
		o.KubeConfigPath = filepath.Join(o.ConfDir, defaultKubeConfigFileName)
	}
	if o.TokenFile == "" {
		o.TokenFile = filepath.Join(o.StateDir, "token")
	}
	if o.CAFile == "" {
		o.CAFile = filepath.Join(o.StateDir, "ca.crt")
	}
	if o.ServiceAccountTokenPath == "" {
		o.ServiceAccountTokenPath = defaults.ServiceAccountTokenPath
	}
	if o.ServiceAccountCAPath == "" {
		o.ServiceAccountCAPath = defaults.ServiceAccountCAPath
	}
	if o.APIServer == "" {
		o.APIServer = kubernetesServiceHost()
	}
	if o.ManagedLabel == "" {
		o.ManagedLabel = defaults.ManagedLabel
	}
	if o.ManagedLabelValue == "" {
		o.ManagedLabelValue = defaults.ManagedLabelValue
	}
	if o.IPTablesPath == "" {
		o.IPTablesPath = defaults.IPTablesPath
	}
	if o.IPSetPath == "" {
		o.IPSetPath = defaults.IPSetPath
	}
	if o.GRPCInboundPort == 0 {
		o.GRPCInboundPort = defaults.GRPCInboundPort
	}
	if requireAPIServer && o.APIServer == "" {
		return fmt.Errorf("kubernetes API server is required")
	}
	return nil
}

func Install(ctx context.Context, opts InstallerOptions) error {
	if err := opts.applyDefaults(true); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, dir := range []string{opts.BinDir, opts.ConfDir, opts.StateDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := copyFileAtomic(opts.SourceBinary, filepath.Join(opts.BinDir, PluginType), 0o755); err != nil {
		return err
	}
	if err := copyFileAtomic(opts.ServiceAccountTokenPath, opts.TokenFile, 0o600); err != nil {
		return err
	}
	if err := copyFileAtomic(opts.ServiceAccountCAPath, opts.CAFile, 0o644); err != nil {
		return err
	}
	if err := writeKubeConfig(opts); err != nil {
		return err
	}
	return installCNIConfig(opts)
}

func InstallLoop(ctx context.Context, opts InstallerOptions, interval time.Duration) error {
	if interval <= 0 {
		interval = defaultInstallerInterval
	}
	if err := Install(ctx, opts); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := Install(ctx, opts); err != nil {
				return err
			}
		}
	}
}

func Uninstall(opts InstallerOptions) error {
	if err := opts.applyDefaults(false); err != nil {
		return err
	}
	if err := uninstallCNIConfig(opts.ConfDir); err != nil {
		return err
	}
	for _, path := range []string{
		filepath.Join(opts.BinDir, PluginType),
		opts.KubeConfigPath,
		opts.TokenFile,
		opts.CAFile,
	} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}

func writeKubeConfig(opts InstallerOptions) error {
	data := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: kubernetes
  cluster:
    certificate-authority: %s
    server: %s
users:
- name: dubbo-cni
  user:
    tokenFile: %s
contexts:
- name: dubbo-cni
  context:
    cluster: kubernetes
    user: dubbo-cni
current-context: dubbo-cni
`, opts.CAFile, opts.APIServer, opts.TokenFile)
	return writeFileAtomic(opts.KubeConfigPath, []byte(data), 0o600)
}

func installCNIConfig(opts InstallerOptions) error {
	target, err := findPrimaryCNIConfig(opts.ConfDir)
	if err != nil {
		return err
	}
	if strings.HasSuffix(target, ".conflist") {
		return patchConflist(target, pluginConfig(opts), true)
	}
	return convertConfToConflist(target, pluginConfig(opts))
}

func uninstallCNIConfig(confDir string) error {
	entries, err := os.ReadDir(confDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read CNI config dir %s: %w", confDir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(confDir, entry.Name())
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if strings.HasSuffix(entry.Name(), installBackupSuffix) {
			original := strings.TrimSuffix(path, installBackupSuffix)
			if strings.HasSuffix(original, ".conf") {
				_ = os.Remove(strings.TrimSuffix(original, ".conf") + ".conflist")
			}
			if err := os.Rename(path, original); err != nil {
				return fmt.Errorf("restore %s: %w", original, err)
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), ".conflist") {
			if err := patchConflist(path, nil, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func findPrimaryCNIConfig(confDir string) (string, error) {
	entries, err := os.ReadDir(confDir)
	if err != nil {
		return "", fmt.Errorf("read CNI config dir %s: %w", confDir, err)
	}
	var conflists, confs []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), installBackupSuffix) {
			continue
		}
		path := filepath.Join(confDir, entry.Name())
		switch {
		case strings.HasSuffix(entry.Name(), ".conflist"):
			conflists = append(conflists, path)
		case strings.HasSuffix(entry.Name(), ".conf"), strings.HasSuffix(entry.Name(), ".json"):
			confs = append(confs, path)
		}
	}
	sort.Strings(conflists)
	sort.Strings(confs)
	for _, path := range append(conflists, confs...) {
		if ok, err := isPrimaryCNIConfig(path); err != nil {
			return "", err
		} else if ok {
			return path, nil
		}
	}
	return "", fmt.Errorf("no primary CNI config found in %s", confDir)
}

func isPrimaryCNIConfig(path string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return false, nil
	}
	if plugins, ok := cfg["plugins"].([]any); ok {
		for _, item := range plugins {
			plugin, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if plugin["type"] == PluginType {
				return true, nil
			}
			if plugin["type"] != "loopback" {
				return true, nil
			}
		}
		return false, nil
	}
	return cfg["type"] != PluginType && cfg["type"] != "loopback", nil
}

func patchConflist(path string, plugin map[string]any, backup bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse CNI conflist %s: %w", path, err)
	}
	plugins, _ := cfg["plugins"].([]any)
	next := make([]any, 0, len(plugins)+1)
	for _, item := range plugins {
		existing, ok := item.(map[string]any)
		if ok && existing["type"] == PluginType {
			continue
		}
		next = append(next, item)
	}
	if plugin != nil {
		next = append(next, plugin)
	}
	cfg["plugins"] = next
	if backup {
		if err := writeBackup(path, raw); err != nil {
			return err
		}
	}
	return writeJSONAtomic(path, cfg, 0o644)
}

func convertConfToConflist(path string, plugin map[string]any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse CNI config %s: %w", path, err)
	}
	backupPath := path + installBackupSuffix
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("backup %s: %w", path, err)
		}
	} else if err != nil {
		return fmt.Errorf("stat %s: %w", backupPath, err)
	}
	conflist := map[string]any{
		"cniVersion": stringValue(cfg["cniVersion"], defaultCNIVersion),
		"name":       stringValue(cfg["name"], strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
		"plugins":    []any{cfg, plugin},
	}
	return writeJSONAtomic(strings.TrimSuffix(path, filepath.Ext(path))+".conflist", conflist, 0o644)
}

func pluginConfig(opts InstallerOptions) map[string]any {
	return map[string]any{
		"type":              PluginType,
		"name":              PluginType,
		"kubernetes":        map[string]any{"kubeconfig": opts.KubeConfigPath},
		"managedLabel":      opts.ManagedLabel,
		"managedLabelValue": opts.ManagedLabelValue,
		"grpcInboundPort":   opts.GRPCInboundPort,
		"stateDir":          opts.StateDir,
		"iptablesPath":      opts.IPTablesPath,
		"ipsetPath":         opts.IPSetPath,
	}
}

func writeBackup(path string, data []byte) error {
	backup := path + installBackupSuffix
	if _, err := os.Stat(backup); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", backup, err)
	}
	return writeFileAtomic(backup, data, 0o644)
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, mode)
}

func copyFileAtomic(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(dst), err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", dst, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return fmt.Errorf("install %s: %w", dst, err)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func kubernetesServiceHost() string {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	if host == "" {
		return ""
	}
	port := os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS")
	if port == "" {
		port = os.Getenv("KUBERNETES_SERVICE_PORT")
	}
	if port == "" {
		port = "443"
	}
	return "https://" + host + ":" + port
}

func stringValue(value any, fallback string) string {
	if s, ok := value.(string); ok && s != "" {
		return s
	}
	return fallback
}
