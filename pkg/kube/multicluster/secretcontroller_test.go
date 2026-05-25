package multicluster

import (
	"errors"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

type recordingHandler struct {
	added   []cluster.ID
	updated []cluster.ID
	deleted []cluster.ID
}

func (h *recordingHandler) clusterAdded(cluster *Cluster) ComponentConstraint {
	h.added = append(h.added, cluster.ID)
	return syncedComponent{}
}

func (h *recordingHandler) clusterUpdated(cluster *Cluster) ComponentConstraint {
	h.updated = append(h.updated, cluster.ID)
	return syncedComponent{}
}

func (h *recordingHandler) clusterDeleted(clusterID cluster.ID) {
	h.deleted = append(h.deleted, clusterID)
}

func (h *recordingHandler) HasSynced() bool {
	return true
}

type syncedComponent struct{}

func (syncedComponent) Close()          {}
func (syncedComponent) HasSynced() bool { return true }

func TestReconcileSecretLifecycle(t *testing.T) {
	controller := newTestController(func([]byte, cluster.ID, ...func(*rest.Config)) (kube.Client, error) {
		return nil, nil
	})
	handler := &recordingHandler{}
	controller.handlers = append(controller.handlers, handler)
	key := types.NamespacedName{Namespace: "dubbo-system", Name: "remote-cluster1"}

	if err := controller.reconcileSecret(key, remoteSecret("cluster1", "kubeconfig-a")); err != nil {
		t.Fatalf("reconcileSecret() add error = %v", err)
	}
	if got := handler.added; len(got) != 1 || got[0] != "cluster1" {
		t.Fatalf("added = %v, want [cluster1]", got)
	}
	if _, ok := controller.cs.Get("cluster1"); !ok {
		t.Fatal("cluster1 not stored after add")
	}

	if err := controller.reconcileSecret(key, remoteSecret("cluster1", "kubeconfig-b")); err != nil {
		t.Fatalf("reconcileSecret() update error = %v", err)
	}
	if got := handler.updated; len(got) != 1 || got[0] != "cluster1" {
		t.Fatalf("updated = %v, want [cluster1]", got)
	}

	controller.deleteSecret(key)
	if got := handler.deleted; len(got) != 1 || got[0] != "cluster1" {
		t.Fatalf("deleted = %v, want [cluster1]", got)
	}
	if _, ok := controller.cs.Get("cluster1"); ok {
		t.Fatal("cluster1 still stored after delete")
	}
}

func TestReconcileSecretRejectsBadRemoteCluster(t *testing.T) {
	controller := newTestController(func([]byte, cluster.ID, ...func(*rest.Config)) (kube.Client, error) {
		return nil, errors.New("bad kubeconfig")
	})
	key := types.NamespacedName{Namespace: "dubbo-system", Name: "remote-cluster1"}
	if err := controller.reconcileSecret(key, remoteSecret("cluster1", "kubeconfig-a")); err == nil {
		t.Fatal("reconcileSecret() error = nil, want bad kubeconfig")
	}
	if _, ok := controller.cs.Get("cluster1"); ok {
		t.Fatal("cluster1 stored despite bad kubeconfig")
	}
}

func TestReconcileSecretRejectsDuplicateClusterID(t *testing.T) {
	controller := newTestController(func([]byte, cluster.ID, ...func(*rest.Config)) (kube.Client, error) {
		return nil, nil
	})
	if err := controller.reconcileSecret(types.NamespacedName{Namespace: "dubbo-system", Name: "remote-a"}, remoteSecret("cluster1", "kubeconfig-a")); err != nil {
		t.Fatalf("reconcileSecret() first add error = %v", err)
	}
	if err := controller.reconcileSecret(types.NamespacedName{Namespace: "dubbo-system", Name: "remote-b"}, remoteSecret("cluster1", "kubeconfig-b")); err == nil {
		t.Fatal("reconcileSecret() duplicate error = nil, want duplicate rejection")
	}
}

func TestParseRemoteClusterSecretValidation(t *testing.T) {
	tests := []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "wrong type",
			secret: &corev1.Secret{
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					RemoteClusterSecretClusterNameKey: []byte("cluster1"),
					RemoteClusterSecretKubeconfigKey:  []byte("config"),
				},
			},
		},
		{
			name: "missing cluster name",
			secret: &corev1.Secret{
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					RemoteClusterSecretKubeconfigKey: []byte("config"),
				},
			},
		},
		{
			name: "missing kubeconfig",
			secret: &corev1.Secret{
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					RemoteClusterSecretClusterNameKey: []byte("cluster1"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parseRemoteClusterSecret(test.secret); err == nil {
				t.Fatal("parseRemoteClusterSecret() error = nil, want validation error")
			}
		})
	}
}

func TestClusterStoreHasSynced(t *testing.T) {
	store := NewClusterStore()
	cluster1 := NewRemoteCluster("cluster1", nil, [32]byte{})
	store.Store(cluster1)
	if store.HasSynced() {
		t.Fatal("HasSynced() = true before cluster sync")
	}
	cluster1.initialSync.Store(true)
	if !store.HasSynced() {
		t.Fatal("HasSynced() = false after cluster sync")
	}
}

func newTestController(builder func([]byte, cluster.ID, ...func(*rest.Config)) (kube.Client, error)) *Controller {
	return &Controller{
		configClusterID:    "primary",
		cs:                 NewClusterStore(),
		secretToCluster:    make(map[types.NamespacedName]cluster.ID),
		clusterToSecret:    make(map[cluster.ID]types.NamespacedName),
		newRemoteClient:    builder,
		startRemoteCluster: func(*Cluster) {},
	}
}

func remoteSecret(clusterName, kubeconfig string) *corev1.Secret {
	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			RemoteClusterSecretClusterNameKey: []byte(clusterName),
			RemoteClusterSecretKubeconfigKey:  []byte(kubeconfig),
		},
	}
}
