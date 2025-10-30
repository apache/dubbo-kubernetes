package apigen

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
)

type APIGenerator struct {
	store model.ConfigStore
}

func NewGenerator(store model.ConfigStore) *APIGenerator {
	return &APIGenerator{
		store: store,
	}
}

func (g *APIGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	resp := model.Resources{}
	return resp, model.DefaultXdsLogDetails, nil
}
