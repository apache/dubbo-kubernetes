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
