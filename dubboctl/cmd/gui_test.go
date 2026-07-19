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

import "testing"

func TestBuildGUIURLs(t *testing.T) {
	tests := []struct {
		name            string
		address         string
		path            string
		wantGUI         string
		wantOverviewURL string
	}{
		{
			name:            "default host and path",
			address:         "http://127.0.0.1:26080",
			path:            "/gui",
			wantGUI:         "http://127.0.0.1:26080/gui/",
			wantOverviewURL: "http://127.0.0.1:26080/gui/api/overview",
		},
		{
			name:            "host without scheme",
			address:         "127.0.0.1:26080",
			path:            "dashboard",
			wantGUI:         "http://127.0.0.1:26080/dashboard/",
			wantOverviewURL: "http://127.0.0.1:26080/dashboard/api/overview",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotGUI, gotOverviewURL, err := buildGUIURLs(test.address, test.path)
			if err != nil {
				t.Fatalf("buildGUIURLs() returned error: %v", err)
			}
			if gotGUI != test.wantGUI {
				t.Fatalf("buildGUIURLs() GUI URL = %q, want %q", gotGUI, test.wantGUI)
			}
			if gotOverviewURL != test.wantOverviewURL {
				t.Fatalf("buildGUIURLs() overview URL = %q, want %q", gotOverviewURL, test.wantOverviewURL)
			}
		})
	}
}
