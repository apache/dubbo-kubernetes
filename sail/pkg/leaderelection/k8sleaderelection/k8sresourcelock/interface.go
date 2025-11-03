package k8sresourcelock

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LeaderElectionRecordAnnotationKey = "control-plane.alpha.kubernetes.io/leader"
)

type LeaderElectionRecord struct {
	HolderIdentity       string      `json:"holderIdentity"`
	HolderKey            string      `json:"holderKey"`
	LeaseDurationSeconds int         `json:"leaseDurationSeconds"`
	AcquireTime          metav1.Time `json:"acquireTime"`
	RenewTime            metav1.Time `json:"renewTime"`
	LeaderTransitions    int         `json:"leaderTransitions"`
}

type Interface interface {
	Get(ctx context.Context) (*LeaderElectionRecord, []byte, error)
	Create(ctx context.Context, ler LeaderElectionRecord) error
	Update(ctx context.Context, ler LeaderElectionRecord) error
	Identity() string
	Key() string
	Describe() string
	RecordEvent(string)
}

type ResourceLockConfig struct {
	Identity      string
	Key           string
	EventRecorder EventRecorder
}
