package rest

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest/v1alpha1"
)

type Resource interface {
	GetMeta() v1alpha1.ResourceMeta
	GetSpec() core_model.ResourceSpec
}
