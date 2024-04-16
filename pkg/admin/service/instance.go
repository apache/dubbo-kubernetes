package service

import (
	"strconv"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

func SearchInstances(rt core_runtime.Runtime, req *model.SearchInstanceReq) ([]*model.SearchInstanceResp, *core_model.Pagination, error) {
	manager := rt.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(rt.AppContext(), dataplaneList, store.ListByNameContains(req.AppName), store.ListByPage(req.PageSize, strconv.Itoa(req.CurPage))); err != nil {
		return nil, nil, err
	}

	res := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		res[i] = &model.SearchInstanceResp{}
		res[i] = res[i].FromDataplaneResource(item)
	}

	return res, &dataplaneList.Pagination, nil
}
