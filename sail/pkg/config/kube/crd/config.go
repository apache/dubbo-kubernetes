package crd

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DubboKind struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              json.RawMessage  `json:"spec"`
	Status            *json.RawMessage `json:"status,omitempty"`
}

func (in *DubboKind) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}

	return nil
}

func (in *DubboKind) DeepCopy() *DubboKind {
	if in == nil {
		return nil
	}
	out := new(DubboKind)
	in.DeepCopyInto(out)
	return out
}

func (in *DubboKind) DeepCopyInto(out *DubboKind) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

func (in *DubboKind) GetObjectMeta() metav1.ObjectMeta {
	return in.ObjectMeta
}

func (in *DubboKind) GetSpec() json.RawMessage {
	return in.Spec
}

func (in *DubboKind) GetStatus() *json.RawMessage {
	return in.Status
}

type DubboObject interface {
	runtime.Object
	GetSpec() json.RawMessage
	GetStatus() *json.RawMessage
	GetObjectMeta() metav1.ObjectMeta
}
