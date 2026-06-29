package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestDashboardManifestFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"prometheus.yaml", "grafana.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: "+name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := dashboardManifestFiles(dir)
	if err != nil {
		t.Fatalf("dashboardManifestFiles() returned error: %v", err)
	}
	want := []string{filepath.Join(dir, "prometheus.yaml"), filepath.Join(dir, "grafana.yaml")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dashboardManifestFiles() = %v, want %v", got, want)
	}
	if dashboardHasTracing(got) {
		t.Fatalf("dashboardHasTracing() = true, want false")
	}
	if names := dashboardWaitDeploymentNames(got); !reflect.DeepEqual(names, []string{"prometheus", "grafana"}) {
		t.Fatalf("dashboardWaitDeploymentNames() = %v", names)
	}
}

func TestDashboardManifestFilesIncludesTracingWhenPresent(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"prometheus.yaml", "grafana.yaml", "tracing.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: "+name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := dashboardManifestFiles(dir)
	if err != nil {
		t.Fatalf("dashboardManifestFiles() returned error: %v", err)
	}
	want := []string{filepath.Join(dir, "prometheus.yaml"), filepath.Join(dir, "grafana.yaml"), filepath.Join(dir, "tracing.yaml")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dashboardManifestFiles() = %v, want %v", got, want)
	}
	if !dashboardHasTracing(got) {
		t.Fatalf("dashboardHasTracing() = false, want true")
	}
	if names := dashboardWaitDeploymentNames(got); !reflect.DeepEqual(names, []string{"prometheus", "grafana", "tracing"}) {
		t.Fatalf("dashboardWaitDeploymentNames() = %v", names)
	}
}

func TestReadManifestObjects(t *testing.T) {
	file := filepath.Join(t.TempDir(), "manifest.yaml")
	manifest := strings.Join([]string{
		"apiVersion: v1",
		"kind: ConfigMap",
		"metadata:",
		"  name: first",
		"---",
		"apiVersion: v1",
		"kind: Service",
		"metadata:",
		"  name: second",
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	objects, err := readManifestObjects(file)
	if err != nil {
		t.Fatalf("readManifestObjects() returned error: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("readManifestObjects() returned %d objects, want 2", len(objects))
	}
	if objects[0].GetKind() != "ConfigMap" || objects[1].GetKind() != "Service" {
		t.Fatalf("unexpected objects: %s, %s", objects[0].GetKind(), objects[1].GetKind())
	}
}

func TestDeploymentAvailable(t *testing.T) {
	replicas := int32(2)
	if deploymentAvailable(&appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &replicas}, Status: appsv1.DeploymentStatus{AvailableReplicas: 1}}) {
		t.Fatal("deploymentAvailable() = true, want false")
	}
	if !deploymentAvailable(&appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &replicas}, Status: appsv1.DeploymentStatus{AvailableReplicas: 2}}) {
		t.Fatal("deploymentAvailable() = false, want true")
	}
}
