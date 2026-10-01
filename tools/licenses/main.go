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

// Package main exports the audited licenses for one binary and platform.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type legalFile struct {
	Path         string   `json:"path"`
	UpstreamFile string   `json:"upstream_file"`
	SHA256       string   `json:"sha256"`
	Package      string   `json:"package,omitempty"`
	CodeFiles    []string `json:"code_files,omitempty"`
}

type component struct {
	ID                string      `json:"id"`
	Version           string      `json:"version"`
	Kind              string      `json:"kind"`
	License           string      `json:"license"`
	LicenseReferences []string    `json:"license_references,omitempty"`
	SourceURL         string      `json:"source_url"`
	RequiredModule    string      `json:"required_module,omitempty"`
	SourceFiles       []string    `json:"source_files,omitempty"`
	Files             []legalFile `json:"files"`
}

type copiedCode struct {
	File      string   `json:"file"`
	Project   string   `json:"project"`
	Copyright []string `json:"copyright"`
}

type manifest struct {
	SchemaVersion int          `json:"schema_version"`
	AuditedDate   string       `json:"audited_date"`
	GoVersion     string       `json:"go_version"`
	Binary        string       `json:"binary,omitempty"`
	Target        string       `json:"target,omitempty"`
	Components    []component  `json:"components"`
	CopiedCode    []copiedCode `json:"copied_code,omitempty"`
}

type goPackage struct {
	ImportPath string
	Dir        string
	Standard   bool
	GoFiles    []string
	SFiles     []string
	HFiles     []string
	CFiles     []string
	EmbedFiles []string
	Module     *struct {
		Path    string
		Version string
		Main    bool
		Replace *struct {
			Path    string
			Version string
		}
	}
}

func main() {
	binary := flag.String("binary", "", "dubboctl or dubbod")
	goos := flag.String("goos", runtime.GOOS, "target operating system")
	goarch := flag.String("goarch", runtime.GOARCH, "target architecture")
	out := flag.String("out", "", "distribution directory")
	flag.Parse()
	if err := export(*binary, *goos, *goarch, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func export(binary, goos, goarch, out string) error {
	mainPackages := map[string]string{"dubboctl": "./dubboctl", "dubbod": "./dubbod/discovery/cmd"}
	mainPackage, ok := mainPackages[binary]
	if !ok || out == "" {
		return fmt.Errorf("-binary dubboctl|dubbod and -out are required")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	output, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if output == root {
		return fmt.Errorf("output must not be the repository root")
	}
	if _, err := os.Stat(filepath.Join(output, "licenses")); !os.IsNotExist(err) {
		return fmt.Errorf("output licenses directory already exists or cannot be inspected")
	}
	data, err := os.ReadFile("licenses/manifest.json")
	if err != nil {
		return err
	}
	var audited manifest
	if err := json.Unmarshal(data, &audited); err != nil {
		return err
	}
	if audited.SchemaVersion != 1 {
		return fmt.Errorf("unsupported license manifest schema %d", audited.SchemaVersion)
	}
	if audited.GoVersion != runtime.Version() {
		return fmt.Errorf("Go toolchain changed from %s to %s; refresh the license audit", audited.GoVersion, runtime.Version())
	}
	cmd := exec.Command("go", "list", "-mod=readonly", "-deps", "-json", mainPackage)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	listed, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list %s dependencies: %w\n%s", binary, err, &stderr)
	}
	modules := map[string]string{}
	packages, codeFiles, localFiles := map[string]bool{}, map[string]bool{}, map[string]bool{}
	decoder := json.NewDecoder(bytes.NewReader(listed))
	for {
		var p goPackage
		if err := decoder.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		packages[p.ImportPath] = true
		if p.Module != nil && !p.Module.Main {
			if p.Module.Replace != nil {
				return fmt.Errorf("replacement for %s must be audited separately: %s@%s", p.Module.Path, p.Module.Replace.Path, p.Module.Replace.Version)
			}
			modules[p.Module.Path] = p.Module.Version
		}
		files := append(append(append(append(append([]string{}, p.GoFiles...), p.SFiles...), p.HFiles...), p.CFiles...), p.EmbedFiles...)
		for _, file := range files {
			if p.Module != nil && p.Module.Main {
				rel, err := filepath.Rel(root, filepath.Join(p.Dir, file))
				if err != nil {
					return err
				}
				localFiles[filepath.ToSlash(rel)] = true
			}
			if p.Standard {
				rel, err := filepath.Rel(filepath.Join(runtime.GOROOT(), "src"), filepath.Join(p.Dir, file))
				if err != nil {
					return err
				}
				codeFiles["std/"+filepath.ToSlash(rel)] = true
			} else {
				codeFiles[p.ImportPath+"/"+filepath.ToSlash(file)] = true
			}
		}
	}
	known := map[string]string{}
	for _, c := range audited.Components {
		if c.Kind == "module" {
			known[c.ID] = c.Version
		}
	}
	for name, version := range modules {
		if known[name] != version {
			return fmt.Errorf("license audit missing or stale for %s@%s; update licenses/manifest.json and upstream texts", name, version)
		}
	}
	selected := manifest{SchemaVersion: 1, AuditedDate: audited.AuditedDate, GoVersion: runtime.Version(), Binary: binary, Target: goos + "-" + goarch}
	fileData := map[string][]byte{}
	notice, err := os.ReadFile("NOTICE")
	if err != nil {
		return err
	}
	// Source attributions are selected separately; dependency NOTICE files are
	// added only when their corresponding packages enter this distribution.
	baseNotice := strings.SplitN(string(notice), "\nThird-party source attributions\n", 2)[0]
	var notices []string
	for _, c := range audited.Components {
		include := c.Kind == "toolchain" || (c.Kind == "module" && modules[c.ID] == c.Version)
		if c.Kind == "copied" {
			include = intersects(c.SourceFiles, localFiles)
		}
		if c.Kind == "data" {
			_, include = modules[c.RequiredModule]
		}
		if !include {
			continue
		}
		copy := c
		copy.Files = nil
		if c.Kind == "toolchain" {
			copy.Version = runtime.Version()
		}
		for _, f := range c.Files {
			if f.Package != "" && !includesPackage(f.Package, packages) {
				continue
			}
			if len(f.CodeFiles) > 0 && !intersects(f.CodeFiles, codeFiles) {
				continue
			}
			clean := filepath.Clean(f.Path)
			if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				return fmt.Errorf("invalid license path %q", f.Path)
			}
			b, err := os.ReadFile(filepath.Join("licenses", clean))
			if err != nil {
				return err
			}
			hash := sha256.Sum256(b)
			if hex.EncodeToString(hash[:]) != f.SHA256 {
				return fmt.Errorf("license checksum mismatch: %s", f.Path)
			}
			if c.Kind == "toolchain" && (f.UpstreamFile == "LICENSE" || f.UpstreamFile == "PATENTS") {
				current, err := os.ReadFile(filepath.Join(runtime.GOROOT(), f.UpstreamFile))
				if err != nil {
					return err
				}
				if !bytes.Equal(current, b) {
					return fmt.Errorf("Go %s changed %s; refresh the license audit", runtime.Version(), f.UpstreamFile)
				}
			}
			if c.Kind == "toolchain" && len(f.CodeFiles) > 0 {
				current, err := os.ReadFile(filepath.Join(runtime.GOROOT(), f.UpstreamFile))
				if err != nil {
					return err
				}
				if !bytes.Contains(current, b) {
					return fmt.Errorf("Go inline license changed: %s", f.UpstreamFile)
				}
			}
			fileData[f.Path] = b
			copy.Files = append(copy.Files, f)
			if strings.HasPrefix(strings.ToUpper(filepath.Base(f.Path)), "NOTICE") {
				notices = append(notices, c.ID+"@"+c.Version+"\n\n"+string(b))
			}
		}
		if strings.Contains(c.License, "MPL-2.0") {
			notices = append(notices, c.ID+"@"+c.Version+" is licensed under MPL-2.0.\n"+
				"The unmodified source code for this component is available at:\n"+c.SourceURL+"\n")
		}
		selected.Components = append(selected.Components, copy)
	}
	for _, c := range selected.Components {
		for _, reference := range c.LicenseReferences {
			if _, ok := fileData[reference]; !ok {
				return fmt.Errorf("missing referenced license %s for %s", reference, c.ID)
			}
		}
	}
	copyrights := map[string]bool{}
	for _, c := range audited.CopiedCode {
		if localFiles[c.File] {
			selected.CopiedCode = append(selected.CopiedCode, c)
			// gRPC's copied buffer already retains its original notice in the
			// source; the module NOTICE is propagated independently.
			if c.Project == "google.golang.org/grpc" {
				continue
			}
			for _, line := range c.Copyright {
				copyrights[line] = true
			}
		}
	}
	var lines []string
	for line := range copyrights {
		lines = append(lines, line)
	}
	sort.Strings(lines)
	if len(lines) > 0 {
		notices = append([]string{"Third-party source attributions\n\n" + strings.Join(lines, "\n") + "\n"}, notices...)
	}
	license, err := os.ReadFile("LICENSE")
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(selected, "", "  ")
	if err != nil {
		return err
	}
	fileData["manifest.json"] = append(encoded, '\n')
	for name, b := range fileData {
		destination := filepath.Join(output, "licenses", name)
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, b, 0644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(output, "LICENSE"), license, 0644); err != nil {
		return err
	}
	finalNotice := strings.TrimSpace(baseNotice) + "\n\n" + strings.Join(notices, "\n--------------------\n\n") + "\n"
	if err := os.WriteFile(filepath.Join(output, "NOTICE"), []byte(finalNotice), 0644); err != nil {
		return err
	}
	fmt.Printf("exported %d components for %s %s/%s\n", len(selected.Components), binary, goos, goarch)
	return nil
}

func intersects(names []string, set map[string]bool) bool {
	for _, name := range names {
		if set[name] {
			return true
		}
	}
	return false
}

func includesPackage(prefix string, packages map[string]bool) bool {
	for name := range packages {
		if name == prefix || strings.HasPrefix(name, prefix+"/") {
			return true
		}
	}
	return false
}
