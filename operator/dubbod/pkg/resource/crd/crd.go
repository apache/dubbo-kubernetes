package crd

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DubboKind struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              json.RawMessage  `json:"spec"`
	Status            *json.RawMessage `json:"status"`
}

func (dk *DubboKind) DeepCopyInto(outDk *DubboKind) {
	*outDk = *dk
	outDk.TypeMeta = dk.TypeMeta
	dk.ObjectMeta.DeepCopyInto(&outDk.ObjectMeta)
	outDk.Spec = dk.Spec
	outDk.Status = dk.Status
}

func (dk *DubboKind) GetObjectMetadata() metav1.ObjectMeta {
	return dk.ObjectMeta
}

func (dk *DubboKind) GetSpec() json.RawMessage {
	return dk.Spec
}

func (dk *DubboKind) GetStatus() *json.RawMessage {
	return dk.Status
}
