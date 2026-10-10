package main

import (
	"testing"
)

func TestSample(t *testing.T) {
	source := []string{"1", "2", "3", "4", "5", "6"}
	sampled := sample(source, 4)
	if len(sampled) != 4 {
		t.Fatalf("expected 4 items, got %d", len(sampled))
	}

	shortSource := []string{"1", "2"}
	sampledShort := sample(shortSource, 4)
	if len(sampledShort) != 2 {
		t.Fatalf("expected 2 items, got %d", len(sampledShort))
	}
}
