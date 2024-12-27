package install

import (
	appsv1 "k8s.io/api/apps/v1"
)

type deployment struct {
	replicaSets *appsv1.ReplicaSet
	deployment  *appsv1.Deployment
}
