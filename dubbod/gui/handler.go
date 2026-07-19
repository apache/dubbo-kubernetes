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

package gui

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"path"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/gui/resources"
)

type Config struct {
	BasePath string `json:"basePath"`
	Product  string `json:"product"`
}

type templateData struct {
	BasePath   string
	ConfigJSON template.JS
}

func NewHandler(basePath string, cfg Config) (http.Handler, error) {
	basePath = NormalizeBasePath(basePath)
	cfg.BasePath = basePath

	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	guiFS := resources.FS()
	tmpl, err := template.ParseFS(guiFS, "index.html")
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, templateData{
		BasePath:   basePath,
		ConfigJSON: template.JS(payload),
	}); err != nil {
		return nil, err
	}

	prefix := basePath
	if prefix != "/" {
		prefix += "/"
	}
	return http.StripPrefix(prefix, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if name != "" && name != "." {
			if _, err := guiFS.Open(name); err == nil {
				http.FileServer(http.FS(guiFS)).ServeHTTP(writer, request)
				return
			}
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(buf.Bytes())
	})), nil
}

func NormalizeBasePath(basePath string) string {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	return strings.TrimRight(trimmed, "/")
}
