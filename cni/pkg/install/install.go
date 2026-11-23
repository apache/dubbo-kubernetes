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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
)

// Config controls how dubbo-cni-node installs the CNI plugin binary and chains CNI config.
type Config struct {
	BinarySource      string
	CNIBinDir         string
	CNINetDir         string
	CNIConfName       string
	SocketPath        string
	LogLevel          string
	FailOpen          bool
	ExcludeNamespaces []string
}

func (c *Config) normalize() {
	if c.BinarySource == "" {
		c.BinarySource = cniconstants.DefaultPluginBinarySource
	}
	if c.CNIBinDir == "" {
		c.CNIBinDir = cniconstants.DefaultCNIBinDir
	}
	if c.CNINetDir == "" {
		c.CNINetDir = cniconstants.DefaultCNINetDir
	}
	if c.SocketPath == "" {
		c.SocketPath = cniconstants.DefaultSocketPath
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// Installer installs and cleans up the chained dubbo-cni plugin on the host node.
type Installer struct {
	cfg Config
}

func New(cfg Config) *Installer {
	cfg.normalize()
	return &Installer{cfg: cfg}
}

// InstallBinary atomically installs the dubbo-cni plugin binary into CNIBinDir.
func (i *Installer) InstallBinary() (string, error) {
	if err := os.MkdirAll(i.cfg.CNIBinDir, 0o755); err != nil {
		return "", fmt.Errorf("create CNI bin dir %s: %w", i.cfg.CNIBinDir, err)
	}
	src, err := os.Open(i.cfg.BinarySource)
	if err != nil {
		return "", fmt.Errorf("open CNI plugin binary %s: %w", i.cfg.BinarySource, err)
	}
	defer src.Close()

	dstPath := filepath.Join(i.cfg.CNIBinDir, cniconstants.PluginName)
	tmpFile, err := os.CreateTemp(i.cfg.CNIBinDir, "."+cniconstants.PluginName+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temp CNI binary: %w", err)
	}
	tmpPath := tmpFile.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmpFile, src); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("copy CNI binary: %w", err)
	}
	if err := tmpFile.Chmod(0o755); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("chmod CNI binary: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close CNI binary: %w", err)
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		return "", fmt.Errorf("install CNI binary to %s: %w", dstPath, err)
	}
	cleanupTmp = false
	return dstPath, nil
}

// InstallConfig patches the active CNI conflist (or upgrades .conf to .conflist)
// to chain dubbo-cni as the last plugin entry.
func (i *Installer) InstallConfig() (string, error) {
	confPath, err := i.resolveConfigFile()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(confPath)
	if err != nil {
		return "", fmt.Errorf("read CNI config %s: %w", confPath, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("parse CNI config %s: %w", confPath, err)
	}

	targetPath := confPath
	var plugins []any
	if existing, ok := doc["plugins"].([]any); ok {
		plugins = existing
	} else {
		// Upgrade single-plugin .conf into a .conflist preserving original plugin as first entry.
		single := map[string]any{}
		for k, v := range doc {
			if k == "cniVersion" || k == "name" {
				continue
			}
			single[k] = v
		}
		cniVersion, _ := doc["cniVersion"].(string)
		if cniVersion == "" {
			cniVersion = "1.0.0"
		}
		name, _ := doc["name"].(string)
		if name == "" {
			name = "k8s-pod-network"
		}
		doc = map[string]any{
			"cniVersion": cniVersion,
			"name":       name,
		}
		plugins = []any{single}
		if strings.HasSuffix(confPath, ".conf") {
			targetPath = confPath + "list"
		}
	}

	entry := map[string]any{
		"type":       cniconstants.PluginName,
		"socketPath": i.cfg.SocketPath,
		"logLevel":   i.cfg.LogLevel,
		"failOpen":   i.cfg.FailOpen,
	}
	if len(i.cfg.ExcludeNamespaces) > 0 {
		entry["excludeNamespaces"] = i.cfg.ExcludeNamespaces
	}

	filtered := make([]any, 0, len(plugins)+1)
	for _, p := range plugins {
		m, ok := p.(map[string]any)
		if ok && m["type"] == cniconstants.PluginName {
			continue
		}
		filtered = append(filtered, p)
	}
	filtered = append(filtered, entry)
	doc["plugins"] = filtered

	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal CNI conflist: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := atomicWrite(targetPath, encoded, 0o644); err != nil {
		return "", err
	}
	if targetPath != confPath {
		_ = os.Remove(confPath)
	}
	return targetPath, nil
}

// Cleanup removes the dubbo-cni entry from all CNI conflist files in CNINetDir.
func (i *Installer) Cleanup() error {
	entries, err := os.ReadDir(i.cfg.CNINetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conflist") {
			continue
		}
		path := filepath.Join(i.cfg.CNINetDir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		plugins, ok := doc["plugins"].([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(plugins))
		changed := false
		for _, p := range plugins {
			m, ok := p.(map[string]any)
			if ok && m["type"] == cniconstants.PluginName {
				changed = true
				continue
			}
			filtered = append(filtered, p)
		}
		if !changed {
			continue
		}
		doc["plugins"] = filtered
		encoded, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		encoded = append(encoded, '\n')
		if err := atomicWrite(path, encoded, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) resolveConfigFile() (string, error) {
	if i.cfg.CNIConfName != "" {
		path := filepath.Join(i.cfg.CNINetDir, i.cfg.CNIConfName)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("CNI config %s not found: %w", path, err)
		}
		return path, nil
	}
	entries, err := os.ReadDir(i.cfg.CNINetDir)
	if err != nil {
		return "", fmt.Errorf("read CNI net dir %s: %w", i.cfg.CNINetDir, err)
	}
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".conflist") || strings.HasSuffix(name, ".conf") {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no .conflist or .conf CNI config found in %s", i.cfg.CNINetDir)
	}
	sort.Strings(candidates)
	return filepath.Join(i.cfg.CNINetDir, candidates[0]), nil
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".dubbo-cni-conf-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
