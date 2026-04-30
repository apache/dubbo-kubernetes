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
