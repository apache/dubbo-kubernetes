package model

import meshconfig "istio.io/api/mesh/v1alpha1"

type PushContext struct {
	Mesh *meshconfig.MeshConfig `json:"-"`
}

func NewPushContext() *PushContext {
	return &PushContext{}
}
