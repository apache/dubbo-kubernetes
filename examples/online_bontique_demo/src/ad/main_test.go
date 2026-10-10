package main

import (
	"testing"
)

func TestGetAdsWithCategory(t *testing.T) {
	resp := GetAds(AdRequest{ContextKeys: []string{"clothing"}})
	if len(resp.Ads) == 0 {
		t.Fatalf("expected ads for category clothing, got empty")
	}
	if resp.Ads[0].RedirectUrl != "/product/66VCHSJNUP" {
		t.Errorf("unexpected ad redirect URL: %s", resp.Ads[0].RedirectUrl)
	}
}

func TestGetAdsRandom(t *testing.T) {
	resp := GetAds(AdRequest{ContextKeys: nil})
	if len(resp.Ads) != 2 {
		t.Fatalf("expected 2 random ads, got %d", len(resp.Ads))
	}
}
