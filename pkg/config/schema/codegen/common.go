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

package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/ast"
	"github.com/apache/dubbo-kubernetes/pkg/env"
)

func Run() error {
	inp, err := buildInputs()
	if err != nil {
		return err
	}

	// Include synthetic types used for XDS pushes
	kindEntries := append([]colEntry{
		{
			Resource: &ast.Resource{Identifier: "Address", Kind: "Address", Version: "internal", Group: "internal"},
		},
		{
			Resource: &ast.Resource{Identifier: "DNSName", Kind: "DNSName", Version: "internal", Group: "internal"},
		},
	}, inp.Entries...)

	sort.Slice(kindEntries, func(i, j int) bool {
		return strings.Compare(kindEntries[i].Resource.Identifier, kindEntries[j].Resource.Identifier) < 0
	})

	// filter to only types agent needs (to keep binary small)
	agentEntries := []colEntry{}
	for _, e := range inp.Entries {
		if strings.Contains(e.Resource.ProtoPackage, "dubbo.apache.org") {
			agentEntries = append(agentEntries, e)
		}
	}

	gvrEntries := append([]colEntry{
		// TODO MCS
	}, inp.Entries...)

	return errors.Join(
		writeTemplate("pkg/config/schema/gvk/resources.gen.go", gvkTemplate, map[string]any{
			"Entries":     inp.Entries,
			"PackageName": "gvk",
		}),
		writeTemplate("pkg/config/schema/gvr/resources.gen.go", gvrTemplate, map[string]any{
			"Entries":     gvrEntries,
			"PackageName": "gvr",
		}),
		writeTemplate("dubbod/discovery/pkg/config/kube/crdclient/types.gen.go", crdclientTemplate, map[string]any{
			"Entries":     inp.Entries,
			"Packages":    inp.Packages,
			"PackageName": "crdclient",
		}),
		writeTemplate("pkg/config/schema/kubetypes/resources.gen.go", typesTemplate, map[string]any{
			"Entries":     inp.Entries,
			"Packages":    inp.Packages,
			"PackageName": "kubetypes",
		}),
		writeTemplate("pkg/config/schema/kubeclient/resources.gen.go", clientsTemplate, map[string]any{
			"Entries":     inp.Entries,
			"Packages":    inp.Packages,
			"PackageName": "kubeclient",
		}),
		writeTemplate("pkg/config/schema/kind/resources.gen.go", kindTemplate, map[string]any{
			"Entries":     kindEntries,
			"PackageName": "kind",
		}),
		writeTemplate("pkg/config/schema/collections/collections.gen.go", collectionsTemplate, map[string]any{
			"Entries":      inp.Entries,
			"Packages":     inp.Packages,
			"PackageName":  "collections",
			"FilePrefix":   "// +build !agent",
			"CustomImport": `  ""`,
		}),
		writeTemplate("pkg/config/schema/collections/collections.agent.gen.go", collectionsTemplate, map[string]any{
			"Entries":      agentEntries,
			"Packages":     inp.Packages,
			"PackageName":  "collections",
			"FilePrefix":   "// +build agent",
			"CustomImport": "",
		}),
	)
}

func writeTemplate(path, tmpl string, i any) error {
	t, err := applyTemplate(tmpl, i)
	if err != nil {
		return fmt.Errorf("apply template %v: %v", path, err)
	}
	dst := filepath.Join(env.DubboSrc, path)
	if err = os.WriteFile(dst, []byte(t), os.ModePerm); err != nil {
		return fmt.Errorf("write template %v: %v", path, err)
	}
	c := exec.Command("goimports", "-w", "-local", "dubbo.apache.org", dst)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func applyTemplate(tmpl string, i any) (string, error) {
	t := template.New("tmpl").Funcs(template.FuncMap{
		"contains": strings.Contains,
	})

	t2 := template.Must(t.Parse(tmpl))

	var b bytes.Buffer
	if err := t2.Execute(&b, i); err != nil {
		return "", err
	}

	return b.String(), nil
}
