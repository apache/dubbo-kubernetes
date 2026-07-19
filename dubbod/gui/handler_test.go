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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBasePath(t *testing.T) {
	cases := map[string]string{
		"":        "/",
		"/":       "/",
		"gui":     "/gui",
		"/gui/":   "/gui",
		" /mesh ": "/mesh",
	}

	for input, want := range cases {
		if got := NormalizeBasePath(input); got != want {
			t.Fatalf("NormalizeBasePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNewHandlerServesAssetsAndFallsBackToIndex(t *testing.T) {
	handler, err := NewHandler("/gui", Config{Product: "Dubbo"})
	if err != nil {
		t.Fatalf("NewHandler() returned error: %v", err)
	}

	assetRequest := httptest.NewRequest(http.MethodGet, "/gui/styles.css", nil)
	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, assetRequest)
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("asset request returned status %d", assetResponse.Code)
	}
	if !strings.Contains(assetResponse.Body.String(), ":root") {
		t.Fatalf("asset response did not contain expected CSS content")
	}

	indexRequest := httptest.NewRequest(http.MethodGet, "/gui/runtime/overview", nil)
	indexResponse := httptest.NewRecorder()
	handler.ServeHTTP(indexResponse, indexRequest)
	if indexResponse.Code != http.StatusOK {
		t.Fatalf("index request returned status %d", indexResponse.Code)
	}
	if !strings.Contains(indexResponse.Body.String(), `<base href="/gui/" />`) {
		t.Fatalf("index response did not render the GUI base path")
	}
}
