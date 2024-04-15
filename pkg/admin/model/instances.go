package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type SearchInstanceReq struct {
	AppName string `form:"appName"`
	PageReq
}

type SearchInstanceResp struct {
	CPU              string            `json:"cpu"`
	DeployCluster    string            `json:"deployCluster"`
	DeployState      State             `json:"deployState"`
	IP               string            `json:"ip"`
	Labels           map[string]string `json:"labels"`
	Memory           string            `json:"memory"`
	Name             string            `json:"name"`
	RegisterClusters []string          `json:"registerClusters"`
	RegisterStates   []State           `json:"registerStates"`
	RegisterTime     string            `json:"registerTime"`
	StartTime        string            `json:"startTime"`
}

func (r *SearchInstanceResp) FromDataplaneResource(dr *mesh.DataplaneResource) *SearchInstanceResp {
	r.IP = dr.GetIP()

	meta := dr.GetMeta()
	r.Name = meta.GetName()
	r.StartTime = meta.GetCreationTime().String()
	r.Labels = meta.GetLabels() // FIXME: in k8s mode, additional labels are append in KubernetesMetaAdapter.GetLabels

	spec := dr.Spec
	{
		statusValue := spec.Extensions[dataplane.ExtensionsPodPhaseKey]
		if v, ok := spec.Extensions[dataplane.ExtensionsPodStatusKey]; ok {
			statusValue = v
		}
		if v, ok := spec.Extensions[dataplane.ExtensionsContainerStatusReasonKey]; ok {
			statusValue = v
		}
		r.DeployState = State{
			Value: statusValue,
		}
	}

	return r
}

type State struct {
	Label string `json:"label"`
	Level string `json:"level"`
	Tip   string `json:"tip"`
	Value string `json:"value"`
}
