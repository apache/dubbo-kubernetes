package cmd

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildRemoteClusterSecret(t *testing.T) {
	secret, err := buildRemoteClusterSecret(remoteSecretArgs{
		clusterName: "cluster2",
		namespace:   "dubbo-system",
	}, []byte("kubeconfig"))
	if err != nil {
		t.Fatalf("buildRemoteClusterSecret() error = %v", err)
	}
	if secret.Name != "dubbo-remote-cluster2" {
		t.Fatalf("secret name = %q, want dubbo-remote-cluster2", secret.Name)
	}
	if got := encodedSecretData(secret, "cluster-name"); got != base64.StdEncoding.EncodeToString([]byte("cluster2")) {
		t.Fatalf("cluster-name data = %q", got)
	}
	if got := encodedSecretData(secret, "kubeconfig"); got != base64.StdEncoding.EncodeToString([]byte("kubeconfig")) {
		t.Fatalf("kubeconfig data = %q", got)
	}
}

func TestRemoteWebhookURLBuildsEnvPath(t *testing.T) {
	got, err := remoteWebhookURL("https://dubbod.example.com", "cluster2", "xds.example.com:26012", "ca.example.com:26012")
	if err != nil {
		t.Fatalf("remoteWebhookURL() error = %v", err)
	}
	want := "https://dubbod.example.com/inject/DUBBO_META_CLUSTER_ID/cluster2/XDS_ADDRESS/xds.example.com:26012/CA_ADDRESS/ca.example.com:26012"
	if got != want {
		t.Fatalf("remoteWebhookURL() = %q, want %q", got, want)
	}
}

func TestBuildRemoteWebhookManifestIncludesCABundle(t *testing.T) {
	manifest, err := buildRemoteWebhookManifest(remoteManifestArgs{
		clusterName: "cluster2",
		webhookURL:  "https://dubbod.example.com",
		xdsAddress:  "xds.example.com:26012",
		caAddress:   "ca.example.com:26012",
		revision:    "default",
	}, []byte("ca-bundle"))
	if err != nil {
		t.Fatalf("buildRemoteWebhookManifest() error = %v", err)
	}
	if len(manifest.Webhooks) != 4 {
		t.Fatalf("webhooks = %d, want 4", len(manifest.Webhooks))
	}
	for _, webhook := range manifest.Webhooks {
		if webhook.ClientConfig.URL == nil || !strings.Contains(*webhook.ClientConfig.URL, "/inject/") {
			t.Fatalf("webhook URL = %v, want remote inject URL", webhook.ClientConfig.URL)
		}
		if string(webhook.ClientConfig.CABundle) != "ca-bundle" {
			t.Fatalf("caBundle = %q, want ca-bundle", string(webhook.ClientConfig.CABundle))
		}
		if len(webhook.Rules) != 2 || webhook.Rules[1].Resources[0] != "services" {
			t.Fatalf("rules = %+v, want pod and service admission rules", webhook.Rules)
		}
	}
}

func TestBuildEastWestGatewayManifest(t *testing.T) {
	gateway, err := buildEastWestGatewayManifest(eastWestGatewayArgs{
		name:        "dubbod-eastwest-gateway",
		namespace:   "dubbo-system",
		serviceType: "NodePort",
		port:        15443,
		targetPort:  15080,
		nodePort:    32443,
		xdsAddress:  "http://192.168.15.164:32010",
	})
	if err != nil {
		t.Fatalf("buildEastWestGatewayManifest() error = %v", err)
	}
	if gateway.Name != "dubbod-eastwest-gateway" {
		t.Fatalf("gateway name = %q", gateway.Name)
	}
	if gateway.Annotations["gateway.dubbo.apache.org/service-type"] != "NodePort" {
		t.Fatalf("service type annotation = %q", gateway.Annotations["gateway.dubbo.apache.org/service-type"])
	}
	if gateway.Annotations["gateway.dubbo.apache.org/xds-address"] != "http://192.168.15.164:32010" {
		t.Fatalf("xds address annotation = %q", gateway.Annotations["gateway.dubbo.apache.org/xds-address"])
	}
	if len(gateway.Spec.Listeners) != 1 || gateway.Spec.Listeners[0].Port != 15443 {
		t.Fatalf("listeners = %#v", gateway.Spec.Listeners)
	}
}
