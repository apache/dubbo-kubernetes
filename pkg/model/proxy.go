package model

import "github.com/apache/dubbo-kubernetes/pkg/cluster"

type NodeMetadata struct {
	Generator string     `json:"GENERATOR,omitempty"`
	ClusterID cluster.ID `json:"CLUSTER_ID,omitempty"`
}
