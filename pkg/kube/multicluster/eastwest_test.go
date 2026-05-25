package multicluster

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
)

func TestParseEastWestGateways(t *testing.T) {
	got, err := ParseEastWestGateways("remote=192.168.15.155:15443,primary=dubbod-eastwest.example.com:443")
	if err != nil {
		t.Fatalf("ParseEastWestGateways() error = %v", err)
	}
	if got[cluster.ID("remote")].Address != "192.168.15.155" || got[cluster.ID("remote")].Port != 15443 {
		t.Fatalf("remote gateway = %#v", got[cluster.ID("remote")])
	}
	if got[cluster.ID("primary")].Endpoint() != "dubbod-eastwest.example.com:443" {
		t.Fatalf("primary endpoint = %q", got[cluster.ID("primary")].Endpoint())
	}
}

func TestParseEastWestGatewaysRejectsBadEntries(t *testing.T) {
	for _, raw := range []string{
		"remote",
		"remote=192.168.15.155",
		"remote=192.168.15.155:notaport",
		"remote=192.168.15.155:0",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseEastWestGateways(raw); err == nil {
				t.Fatal("ParseEastWestGateways() returned nil error")
			}
		})
	}
}
