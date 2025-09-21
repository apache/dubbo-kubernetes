package crd

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DubboKind struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              json.RawMessage  `json:"spec"`
	Status            *json.RawMessage `json:"status,omitempty"`
}
